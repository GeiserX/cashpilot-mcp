#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const fs = require("fs");
const { install, BIN_PATH } = require("./postinstall.js");

async function main() {
  // npm 12, pnpm and Bun skip postinstall by default, so the first run
  // downloads (and verifies) the binary itself.
  if (!fs.existsSync(BIN_PATH)) {
    await install();
  }
  const env = Object.assign({}, process.env, { TRANSPORT: "stdio" });
  try {
    execFileSync(BIN_PATH, process.argv.slice(2), { env, stdio: "inherit" });
  } catch (err) {
    process.exit(err.status || 1);
  }
}

main().catch((err) => {
  process.stderr.write(`cashpilot-mcp: ${err.message}\n`);
  process.exit(1);
});
