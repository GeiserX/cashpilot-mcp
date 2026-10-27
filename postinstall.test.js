"use strict";

const test = require("node:test");
const assert = require("node:assert");
const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { Readable } = require("stream");
const { getAssetName, expectedChecksum, nextUrl, saveStream } = require("./postinstall.js");

const version = require("./package.json").version;

function tmpdir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "cashpilot-mcp-test-"));
}

test("asset names match the goreleaser archives", () => {
  assert.strictEqual(getAssetName("linux", "x64"), `cashpilot-mcp_${version}_linux_amd64.tar.gz`);
  assert.strictEqual(getAssetName("win32", "arm64"), `cashpilot-mcp_${version}_windows_arm64.zip`);
  assert.throws(() => getAssetName("aix", "ppc64"), /Unsupported platform/);
});

test("the checksum comes from the checksums.txt bundled in the package", () => {
  const dir = tmpdir();
  assert.throws(() => expectedChecksum("a.tar.gz", dir), /checksums.txt is missing/);

  const hash = "a".repeat(64);
  fs.writeFileSync(path.join(dir, "checksums.txt"), `${"b".repeat(64)}  other.tar.gz\n${hash}  a.tar.gz\n`);
  assert.strictEqual(expectedChecksum("a.tar.gz", dir), hash);
  assert.throws(() => expectedChecksum("missing.tar.gz", dir), /no entry/);
});

test("redirects must stay on https", () => {
  assert.strictEqual(nextUrl("https://github.com/x/y", "/z"), "https://github.com/z");
  assert.strictEqual(
    nextUrl("https://github.com/x", "https://objects.githubusercontent.com/a"),
    "https://objects.githubusercontent.com/a"
  );
  assert.throws(() => nextUrl("https://github.com/x", "http://evil.example/a"), /non-https/);
  assert.throws(() => nextUrl("http://github.com/x", "http://github.com/x"), /non-https/);
});

test("downloads are hashed while streamed to disk", async () => {
  const dest = path.join(tmpdir(), "out");
  const data = Buffer.from("binary bytes");
  const digest = await saveStream(Readable.from([data]), dest);
  assert.strictEqual(digest, crypto.createHash("sha256").update(data).digest("hex"));
  assert.deepStrictEqual(fs.readFileSync(dest), data);
});

test("downloads past the size cap are refused and removed", async () => {
  const dest = path.join(tmpdir(), "out");
  await assert.rejects(saveStream(Readable.from([Buffer.alloc(10), Buffer.alloc(10)]), dest, 15), /larger than 15 bytes/);
  assert.strictEqual(fs.existsSync(dest), false);
});

test("a failed download rejects even when the destination cannot be written", async () => {
  const dest = path.join(tmpdir(), "missing-dir", "out");
  await assert.rejects(saveStream(Readable.from([Buffer.alloc(10)]), dest, 100));
});
