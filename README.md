<div align="center">

# Kept

**When the meeting's over, you're done.**

The meeting notetaker for Apple Silicon Macs. It records, transcribes, and writes down what
actually matters — all on your own machine. The audio never leaves it.

[**kept.lixiaolong.me**](https://kept.lixiaolong.me) · [Changelog](https://kept.lixiaolong.me/changelog/) · [中文说明](./README.zh-CN.md)

</div>

---

> **This repository is for issues and questions.** Kept's source lives in a private repo — what
> you'll find here is the README above and the issue tracker below. If something is broken, missing,
> or confusing, [open an issue](https://github.com/xlsama/kept/issues) and it lands in front of the
> person who wrote the code.

## What it does

**It starts itself.** Join a call and a small bar appears in the corner. Ten seconds after the last
person leaves, it stops and starts transcribing. You don't take a single note.

**It runs on your Mac.** FireRedASR2 for Chinese, Parakeet for English. Which second each word lands
on is measured from the audio itself, not estimated from word count — click any word and playback
jumps there. Mixed Chinese and English stays intact.

**The notes are usable.** Most AI notes just tell you the meeting again; the person receiving them
ends up asking another AI to shorten it. Kept writes outcomes only: what was decided, who owns it,
when it's due, what's still open. Don't trust a line? Click it and land on the exact words.

**It knows who is who.** Plenty of tools drop a twenty-person call into one single "speaker". Kept
separates everyone, and once you enroll a colleague's voice their name shows up in every meeting
after that. Below the confidence threshold it writes "Speaker 1" rather than guessing.

Also: three recording sources (mic / mic + one app / mic + everything), audio and video import,
lossless two-track FLAC with echo cancellation, full-text search across every meeting, export to
Markdown / TXT / mixed m4a, an MCP server so Claude Code and Codex can read your meetings
(read-only, text-only, off by default), and an ⌥⌘R global hotkey.

## Privacy

Apart from these four things, nothing goes out. The same list is printed inside the app, so you can
sit with a firewall and count:

1. License activation, renewal, deactivation
2. Checking whether a new version exists
3. Downloading the speech models (HuggingFace or a mirror)
4. The AI service *you* configured — text only, and only after you approve it

No cloud transcription. No bot joining your call. No telemetry. Audio, voiceprints and screen
content never leave the machine.

## Accuracy

| What | Number | Measured on |
|---|---|---|
| Chinese meetings, character error rate | 9.3% | our own set of real meetings |
| Speaker attribution accuracy | 88.0% (en) / 82.2% (zh) | 25 sessions, 15.2 h, hand-labelled (AMI / AISHELL-4) |
| Speaker verification EER | 0.26% | VoxCeleb1-O |

Every release is measured again, and the method, definitions and raw results are published with it.
A new model only ships when it wins on two independent measures — one is not enough.

A one-hour meeting takes about 12 minutes to process, speakers included, without going online.

## Requirements

- macOS 15 or later
- Apple Silicon (no Intel Mac or Windows build — on-device transcription leans on the Neural Engine
  and unified memory, and we'd rather not ship a version that technically runs but is miserable)
- ~1 GB for the speech models, downloaded on first launch

## Pricing

$49.9 once — 12 months of updates, your own 2 Macs, 30-day refund, every feature in one edition.
From month 13 renewal is optional and only affects access to new versions; skip it and your build
keeps working forever. AI notes run on your own API key, so that cost goes straight to the provider.

Payments and invoicing are handled by Creem as merchant of record.

## Reporting something

[Open an issue.](https://github.com/xlsama/kept/issues) Useful things to include for a bug:

- Kept version (Settings › General › Current version) and your macOS version
- What you did, what you expected, what happened instead
- If it's about transcription or notes: which language, roughly how long the meeting was

Please don't paste meeting content you'd rather not make public — this tracker is world-readable.
If something is sensitive, mail <im.xlsama@gmail.com> instead.
