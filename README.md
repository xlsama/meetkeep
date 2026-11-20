# meetkeep-poc

MyGo 0.10 上的会议转写 POC：Swift native module 采麦克风和系统音频，Go 按停顿切句后调 Azure MAI-Transcribe-2，React 页面实时显示。

```sh
bun install
AZURE_SPEECH_ENDPOINT=https://<resource>.cognitiveservices.azure.com AZURE_SPEECH_KEY=<key> bun run dev
bun run dev:op   # 或者从 1Password 取 key
bun run check && bun run test
```

需要 macOS 14.2+、Go 1.27+、Bun、Xcode。Windows 上应用能构建运行，但还没有采集实现。
