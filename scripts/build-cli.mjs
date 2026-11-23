import { execFileSync } from "node:child_process";

const env = (name) => execFileSync("go", ["env", name], { encoding: "utf8" }).trim();
const platform = `${env("GOOS")}-${env("GOARCH")}`;
const out = platform.startsWith("windows") ? `resources/${platform}/bin/` : `resources/${platform}/`;

execFileSync("go", ["build", "-trimpath", "-ldflags=-s -w", "-o", out, "./cmd/meetkeep"], { stdio: "inherit" });
