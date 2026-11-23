import { defineConfig } from "mygo-cli";
import pkg from "./package.json" with { type: "json" };

declare const process: { env: Record<string, string | undefined> };

export default defineConfig({
  name: "MeetKeep",
  identifier: "app.meetkeep.open",
  version: pkg.version,
  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web && bun run build:cli",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
  macos: {
    // The identity that signs the app, such as "Developer ID Application:
    // Jane Doe (TEAMID)": .env sets it. Without one the app is signed ad hoc,
    // and macOS asks for the microphone again after every build.
    ...(process.env.MEETKEEP_SIGNING_IDENTITY ? { signingIdentity: process.env.MEETKEEP_SIGNING_IDENTITY } : {}),
    // The release workflow keeps the notary's credentials in a keychain of
    // its own, and names it here.
    ...(process.env.NOTARY_KEYCHAIN
      ? { notarize: { keychainProfile: "notary", keychain: process.env.NOTARY_KEYCHAIN } }
      : {}),
    infoPlist: { LSEnvironment: { MallocLargeCache: "0" } },
  },
  windows: { installerInclude: "scripts/installer.nsh" },
  // Apps update themselves from the releases of the repository, signed
  // with the key whose public half this is; the release workflow has the
  // secret half.
  updates: {
    publicKey: "W/AFTn23GjxPbPc8h6bbzzQu/x3wBvbgRHbS65HNTz4=",
    github: "xlsama/meetkeep",
  },
  // Phones: `mygo dev -platform ios` and `-platform android` run the app on a
  // simulator, an emulator or a device (`mygo devices` lists them, and
  // `mygo doctor` tells what is missing). What devices and stores need:
  // ios: { team: "ABCDE12345", infoPlist: { NSCameraUsageDescription: "Scan receipts." } },
  // android: { keystore: "upload.jks", keyAlias: "upload", permissions: ["CAMERA"] },
});
