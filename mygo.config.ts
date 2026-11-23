import { defineConfig } from "mygo-cli";

declare const process: { env: Record<string, string | undefined> };

export default defineConfig({
  name: "MeetKeep",
  identifier: "app.meetkeep.open",
  version: "0.1.0",
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
    infoPlist: { LSEnvironment: { MallocLargeCache: "0" } },
  },
  // Phones: `mygo dev -platform ios` and `-platform android` run the app on a
  // simulator, an emulator or a device (`mygo devices` lists them, and
  // `mygo doctor` tells what is missing). What devices and stores need:
  // ios: { team: "ABCDE12345", infoPlist: { NSCameraUsageDescription: "Scan receipts." } },
  // android: { keystore: "upload.jks", keyAlias: "upload", permissions: ["CAMERA"] },
});
