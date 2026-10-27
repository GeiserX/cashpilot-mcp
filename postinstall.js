#!/usr/bin/env node
"use strict";

// Downloads the cashpilot-mcp release binary for this platform and checks it
// against checksums.txt, which the release workflow puts INSIDE the npm
// package. A checksum fetched from the same GitHub release as the binary
// would only catch corruption, not a swapped asset.
//
// Runs as the postinstall script, and again from run.js on first use, because
// npm 12, pnpm and Bun skip install scripts by default.
//
// Progress goes to stderr: when run.js calls this, stdout is the MCP stream.

const { execFileSync } = require("child_process");
const crypto = require("crypto");
const fs = require("fs");
const https = require("https");
const path = require("path");

const VERSION = require("./package.json").version;
const REPO = "GeiserX/cashpilot-mcp";
const BIN_NAME = process.platform === "win32" ? "cashpilot-mcp.exe" : "cashpilot-mcp";
const BIN_DIR = path.join(__dirname, "bin");
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);

const MAX_REDIRECTS = 10;
const REQUEST_TIMEOUT_MS = 30_000;
// The archives are about 5 MiB; anything far larger is not ours.
const MAX_DOWNLOAD_BYTES = 64 * 1024 * 1024;

function log(msg) {
  process.stderr.write(msg + "\n");
}

function getAssetName(platform = process.platform, arch = process.arch) {
  const platformMap = { darwin: "darwin", linux: "linux", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };
  const p = platformMap[platform];
  const a = archMap[arch];
  if (!p || !a) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }
  const ext = p === "windows" ? "zip" : "tar.gz";
  return `cashpilot-mcp_${VERSION}_${p}_${a}.${ext}`;
}

// expectedChecksum reads the sha256 of assetName from the checksums.txt that
// shipped in the package.
function expectedChecksum(assetName, dir = __dirname) {
  const file = path.join(dir, "checksums.txt");
  if (!fs.existsSync(file)) {
    throw new Error("checksums.txt is missing from the package; refusing to install an unverified binary");
  }
  for (const line of fs.readFileSync(file, "utf-8").split("\n")) {
    const [hash, name] = line.trim().split(/\s+/);
    if (name === assetName && /^[a-f0-9]{64}$/.test(hash)) {
      return hash;
    }
  }
  throw new Error(`checksums.txt has no entry for ${assetName}`);
}

// nextUrl resolves a redirect and refuses anything that is not https.
function nextUrl(base, location) {
  const u = new URL(location, base);
  if (u.protocol !== "https:") {
    throw new Error(`Refusing non-https URL: ${u.protocol}//${u.host}`);
  }
  return u.href;
}

// saveStream writes a response to dest, hashing it on the way and aborting
// past maxBytes. It resolves to the sha256 hex digest.
function saveStream(stream, dest, maxBytes = MAX_DOWNLOAD_BYTES) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash("sha256");
    const out = fs.createWriteStream(dest, { mode: 0o600 });
    let size = 0;
    let failed = false;
    const fail = (err) => {
      if (failed) return;
      failed = true;
      stream.destroy();
      out.destroy();
      fs.rmSync(dest, { force: true });
      reject(err);
    };
    stream.on("data", (chunk) => {
      size += chunk.length;
      if (size > maxBytes) {
        fail(new Error(`Download is larger than ${maxBytes} bytes; refusing it`));
        return;
      }
      hash.update(chunk);
    });
    stream.on("error", fail);
    out.on("error", fail);
    out.on("finish", () => {
      if (!failed) resolve(hash.digest("hex"));
    });
    stream.pipe(out);
  });
}

function download(url, dest, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > MAX_REDIRECTS) {
      return reject(new Error(`Too many redirects (>${MAX_REDIRECTS})`));
    }
    let target;
    try {
      target = nextUrl(url, url);
    } catch (err) {
      return reject(err);
    }
    const req = https.get(target, { timeout: REQUEST_TIMEOUT_MS }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        let next;
        try {
          next = nextUrl(target, res.headers.location);
        } catch (err) {
          return reject(err);
        }
        return download(next, dest, redirects + 1).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        res.resume();
        return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
      }
      saveStream(res, dest).then(resolve, reject);
    });
    req.on("timeout", () => {
      req.destroy(new Error(`Request timed out after ${REQUEST_TIMEOUT_MS}ms`));
    });
    req.on("error", reject);
  });
}

// extract unpacks the binary into a fresh directory, then renames it into
// place, so a concurrent run never sees a half-written file. No shell: every
// path is passed as an argument, never interpolated into a command line.
function extract(archive, assetName) {
  const work = fs.mkdtempSync(path.join(BIN_DIR, ".extract-"));
  try {
    if (assetName.endsWith(".zip")) {
      if (process.platform === "win32") {
        execFileSync(
          "powershell",
          [
            "-NoProfile",
            "-NonInteractive",
            "-Command",
            "Expand-Archive -Force -LiteralPath $env:CP_ARCHIVE -DestinationPath $env:CP_DEST",
          ],
          { stdio: "ignore", env: { ...process.env, CP_ARCHIVE: archive, CP_DEST: work } }
        );
      } else {
        execFileSync("unzip", ["-o", archive, BIN_NAME, "-d", work], { stdio: "ignore" });
      }
    } else {
      execFileSync("tar", ["-xzf", archive, "-C", work, BIN_NAME], { stdio: "ignore" });
    }
    const extracted = path.join(work, BIN_NAME);
    if (process.platform !== "win32") {
      fs.chmodSync(extracted, 0o755);
    }
    fs.renameSync(extracted, BIN_PATH);
  } finally {
    fs.rmSync(work, { recursive: true, force: true });
  }
}

async function install() {
  if (fs.existsSync(BIN_PATH)) {
    return BIN_PATH;
  }
  const assetName = getAssetName();
  const expected = expectedChecksum(assetName);
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${assetName}`;

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const archive = path.join(BIN_DIR, `.download-${process.pid}-${assetName}`);
  log(`Downloading cashpilot-mcp v${VERSION} for ${process.platform}-${process.arch}...`);
  try {
    const actual = await download(url, archive);
    if (actual !== expected) {
      throw new Error(`Checksum mismatch for ${assetName}: expected ${expected}, got ${actual}`);
    }
    log("SHA-256 checksum verified against the checksums.txt shipped in the package.");
    extract(archive, assetName);
  } finally {
    fs.rmSync(archive, { force: true });
  }
  log(`Installed cashpilot-mcp to ${BIN_PATH}`);
  return BIN_PATH;
}

module.exports = { install, BIN_PATH, getAssetName, expectedChecksum, nextUrl, saveStream };

if (require.main === module) {
  install().catch((err) => {
    log(`Failed to install cashpilot-mcp: ${err.message}`);
    process.exit(1);
  });
}
