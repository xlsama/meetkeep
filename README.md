<p align="center">
  <img src="screenshots/logo.png" alt="Kept" width="96" height="96"/>
</p>

<p align="center">
  <strong>Kept</strong><br/>
  When the meeting's over, you're done.
</p>

<p align="center">
  The meeting notetaker for Apple Silicon Macs. It records, transcribes, and writes down what actually matters — all on your own machine. The audio never leaves.
</p>

<p align="center">
  <a href="https://keptmac.com"><strong>keptmac.com</strong></a>
  ·
  <a href="https://keptmac.com/changelog/">Changelog</a>
  ·
  <a href="https://github.com/xlsama/kept/issues">Issues</a>
  ·
  <a href="./README.zh-CN.md">中文</a>
</p>

---

<img src="screenshots/en-hero.webp" alt="Kept meeting view: word-level transcript on the left, AI notes on the right" width="100%"/>

<img src="screenshots/en-summary.webp" alt="Kept visual summary: takeaways, decisions, and action items on one page" width="100%"/>

## Features

* 💻 Native macOS app. Written in Swift. Built for Apple Silicon.
* 🎙️ **Starts itself** — Join a call and recording begins. About ten seconds after the last person leaves, it stops and starts transcribing.
* 🔒 **Runs on your Mac** — FireRedASR2 for Chinese, Parakeet for English. Word-level timestamps from the audio. Mixed Chinese and English stay intact.
* 📝 **Notes you can use** — Decisions, owners, due dates, open questions. Click any line to hear the original words. One-page visual summary for chat or updates.
* 👥 **Knows who is who** — Separates speakers; enroll a voice once and names carry across meetings.
* 🎛️ Three recording sources: mic / mic + one app / mic + system audio. Import audio and video too.
* 🔍 Full-text search across every meeting. Export to Markdown, TXT, or mixed m4a.
* 🔌 Optional MCP server so Claude Code and Codex can read your meetings (read-only, text-only, off by default).
* 🍎 Requires macOS 15+ and Apple Silicon.

## Privacy

Transcription and speaker ID run on-device. Audio is never uploaded. Aside from these four cases, Kept does not reach the network:

1. License activation, renewal, deactivation
2. Checking for updates
3. Downloading speech models (HuggingFace or a mirror)
4. The AI provider *you* configure — text only, after you approve

No cloud transcription. No bot joins your call. No telemetry.

## Download

* [Download for Mac](https://keptmac.com)

## Issues

This public repo is for [issues and questions](https://github.com/xlsama/kept/issues). Source lives elsewhere.

Useful context for a bug report:

* Kept version (Settings › General) and macOS version
* What you did, what you expected, what happened
* For transcription or notes: language and roughly how long the meeting was

Please don't paste private meeting content — this tracker is public. Sensitive matters: <im.xlsama@gmail.com>

---

Kept is built for people who sit in too many meetings and still want the notes to be useful. If it saves you a few hours a week, a license keeps the work going ❤️
