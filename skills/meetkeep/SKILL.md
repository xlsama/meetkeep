---
name: meetkeep
description: Read the user's meetings from MeetKeep, the meeting recorder on their machine - which meetings they had, what each one concluded, its todos (action items), open questions, topics and timeline, and who said what - and turn an audio or video file into a meeting with a transcript and notes. Use when the user asks what meetings they had today or last week, what a meeting decided, what their todos are, who brought something up, what exactly was said, or asks to transcribe or summarize a recording; also when a daily report, weekly report or review needs what was said in meetings, even if MeetKeep is not named. The data stays on the machine; query it with the meetkeep CLI.
---

# meetkeep

MeetKeep is the meeting recorder on the user's machine: it records a meeting, transcribes it, tells the speakers apart and writes notes. The `meetkeep` CLI reads its database and changes nothing. The one exception is `import`, which hands a file to the running MeetKeep app to process.

Output is text to read; `--json` prints JSON instead. On failure the exit code is not 0 and the message is on stderr (`{"error": "..."}` with `--json`). `meetkeep <command> --help` has the flags of a command.

## Pick the right command

An hour of meeting is about 1,000 to 5,000 characters of notes and 10,000 to 20,000 of transcript. Read the notes first, and take the stretch of transcript you need when the exact words matter. Do not read a whole transcript into context.

| To answer | Use |
|---|---|
| Which meetings were there | `meetkeep list --date yesterday`, `meetkeep list --from -7d`: ID, start, length, title, a one-line conclusion and the speakers of each |
| Meetings with someone | `meetkeep list --person 陈昊` |
| What a meeting decided, and who is to do what | `meetkeep show <meeting>`: conclusions, todos, open questions, the points of each topic, and a timeline of chapters with a sentence or two each |
| What exactly was said | `meetkeep transcript <meeting> --chapter 3` (chapter 3 of the timeline), `--around 30:13` (2 minutes before and after; `--window 5m` changes that), `--from 12:00 --to 20:00`, `--speaker 陈昊` |
| Which meeting mentions something | `meetkeep search 标书`: the matching notes and transcript lines carry their moments; follow up with `transcript <meeting> --around <moment>` |
| Todos across meetings | `meetkeep todos` (unchecked ones of the last 30 days), `--owner 我`, `--from -7d`, `--all` to include checked ones |
| Who is in the meetings | `meetkeep people`: the named speakers and how many meetings each speaks in |
| Turn a recording into a meeting | `meetkeep import <file>`, see below |
| How far processing got | `meetkeep status <meeting>`; without a meeting, every meeting that is not done |
| Why processing is slow or failed | `meetkeep steps <meeting>`: time, memory and error of each step |

`<meeting>` is an ID (such as `20261123-085815-c56a`), the start of one, or `latest` (the meeting that began last).

## Times

- `--date`, `--from` and `--to` pick meetings by when they began: `today`, `yesterday`, `2026-11-20`, `2026-11-20T15:00`, `-7d` (the start of the day 7 days ago), `-3h`, `-30m`. A `--to` that names a day includes that day. Without a range every meeting counts (`todos`: the last 30 days).
- A moment of a meeting is `12:30`, `1:02:03` or seconds. The `[30:13]` in the output is such a moment: the one after a line of the notes is the sentence that line rests on, and goes straight into `--around`.
- In JSON, times are ISO 8601 in the local time zone, the length of a meeting is in minutes, and moments of a meeting also come in milliseconds (`atMs`, `startMs`).

## Processing a recording

The MeetKeep app has to be running (in the menu bar is enough). The file goes through what importing it in the app does: transcription, telling speakers apart, polishing, notes. That takes minutes.

```sh
meetkeep import ~/Downloads/interview.wav     # returns at once, with the meeting's ID
meetkeep wait <ID> --timeout 9m               # until it is processed; exits non-zero on timeout: run it again
meetkeep show <ID>                            # the notes
meetkeep transcript <ID> --chapter 1          # the words, when needed
```

- `meetkeep import <file> --wait` does the first two in one. Keep the wait shorter than the timeout of your own command; after a timeout, `meetkeep wait <ID>` goes on waiting and `meetkeep status <ID>` tells the progress (such as `processing: transcribe 40%`).
- `MeetKeep is not running`: ask the user to open the app, or run `open -a MeetKeep`, then try again. A status of `stalled` has the same cause; the app resumes the meeting once it is open.
- Common audio and video formats work: wav, mp3, m4a, flac, ogg, opus, mp4, mov, mkv, webm and others.
- An imported meeting is titled after its file until the notes give it a title, and begins when the file was last modified.

## Reading the results

- The notes are written by a model from the transcript. Before quoting someone, or relying on a number or a date, check the sentence with `transcript`.
- When the user edited the notes, `show` prints their version (`edited` is true in JSON), and a todo marked `[x]` is one they checked off.
- Speakers carry the names the user gave them. A name like `说话人 2` is a speaker nobody has named, and the same number in another meeting is another person. `我` is the user (the microphone). `(uncertain)` after a name means the sentence may be someone else's.
- The transcript is the polished one: fillers removed and misheard names set right. `--raw` gives the words as they were said.
- A meeting that is not `done`: while it is `recording` or `processing`, the transcript is the draft kept during the meeting, without speakers (only `我` and `对方`), and the notes are a running outline or missing. Wait until it is processed. `status` tells why a meeting `failed`.
- The meetings and their notes are in the language they were held in.
- No meeting here does not mean there was none: only meetings recorded or imported with MeetKeep are kept.
