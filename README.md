# MeetKeep

开源的桌面会议记录应用：录制会议，实时转写，生成纪要和白板。macOS 和 Windows 双端，数据存本地。用 [MyGo](https://github.com/egoist/mygo) 写成：界面是 React，业务是 Go，采集和语音模型是 Swift 和 Rust 的原生模块。

范围和阶段见 `docs/plan.md`。

```sh
bun install
bun run dev
bun run check && bun run test
```

`bun run build` 默认做 ad-hoc 签名，每次构建后 macOS 都会重新要麦克风和录音权限。在 `.env` 里写 `MEETKEEP_SIGNING_IDENTITY="Developer ID Application: 名字 (TEAMID)"`（`security find-identity -v -p codesigning` 列出可用的身份）就用这个身份签名，权限跟着身份走，重装和更新后不用重新授权。

macOS 上需要 macOS 14.2+、Go 1.27+、Bun、Xcode、Rust；Windows（amd64）上需要 Go、Bun、Rust（MSVC）。Windows 没有真机，靠 CI 验证：`.github/workflows/windows.yml`（手动触发）跑测试，并在虚拟声卡上把应用实际跑一遍、留下每一步的截图。现状和没验证的部分记在 `docs/plan.md`。

## 发版

```sh
bun run release
```

`bumpp` 改 `package.json` 里的版本号（`mygo.config.ts` 读它），提交、打 `v<版本>` 的 tag 并推送。tag 触发 `.github/workflows/release.yml`：macOS（arm64）和 Windows（amd64）各自构建，`.dmg` 和安装程序 `-setup-*.exe` 传到这个 tag 的 GitHub Release 并发布。在 Actions 里手动触发同一个 workflow 只构建、不发版，产物留在这次运行里。

仓库里配了 `MACOS_CERTIFICATE`、`MACOS_CERTIFICATE_PASSWORD`、`MACOS_SIGNING_IDENTITY` 这几个 secret 时，macOS 的包用 Developer ID 签名；再配 `APPLE_ID`、`APPLE_TEAM_ID`、`APPLE_APP_PASSWORD` 就同时公证。没配时是 ad-hoc 签名，别的 Mac 上会被 Gatekeeper 拦。Windows 的包没有签名，SmartScreen 会提示。

转写用 Azure 的 MAI-Transcribe-2，或自定义的 OpenAI 兼容服务（`/audio/transcriptions`，自建的也行）；纪要和白板的模型在 Claude、DeepSeek、OpenRouter 和自定义的 OpenAI 兼容服务之间切换，各存各的密钥和模型名，都在应用的设置页里填。自定义服务可以加多个，各起一个名字，在「服务」下拉里和内置的排在一起。设置存在 `~/.config/meetkeep/settings.json`，会议存在同目录的 `meetkeep.db`，录音默认存在 `~/Documents/MeetKeep`。

每次处理的每一步都记在 `meetkeep.db` 的 `steps` 表里：排队、导入、转写、混音、分人、声纹、纪要各用了多久、内存峰值多少、失败原因和这一步的数据量。`meetkeep steps <会议 ID>` 按处理的次数列出来。

## 命令行和 skill

`meetkeep` 是给 Agent 和脚本用的命令行（`cmd/meetkeep`），随应用一起打包，在「设置 → Agent」里安装到 `~/.local/bin`（Windows 上是 `%LOCALAPPDATA%\Microsoft\WindowsApps\meetkeep.cmd`，这个目录默认在 PATH 里）。它只读数据库；`import` 经应用监听的 socket（`~/.config/meetkeep/meetkeep.sock`）把文件交给运行中的应用，走和界面里导入相同的流程。输出默认是文本，`--json` 是 JSON，都是英文。

```sh
meetkeep list --from -7d                    # 最近 7 天的会议
meetkeep show latest                        # 纪要：结论、待办、还没定的事、话题、时间轴
meetkeep transcript latest --around 30:13   # 这个时刻前后的原话
meetkeep search 标书                        # 标题、纪要、逐字稿里找
meetkeep todos --owner 我                   # 跨会议的待办
meetkeep import ~/Downloads/访谈.wav --wait # 交给应用整理，等它完成
meetkeep status <会议 ID>                   # 整理到哪一步了
```

`skills/meetkeep` 是教 Agent 用它的 skill，用 `npx skills add xlsama/meetkeep --skill meetkeep -g` 安装（设置页里可以复制这条命令），`meetkeep skill show` 打印同一份内容。开发时用 `go run ./cmd/meetkeep`，`bun run build:cli` 只构建命令行。

## 结构

| 位置 | 内容 |
|---|---|
| `main.go`、`windows.go`、`services.go`、`events.go` | 应用入口、窗口和菜单栏、给界面调用的服务和事件 |
| `cmd/meetkeep`、`skills/meetkeep` | 命令行和教 Agent 用它的 skill |
| `internal/recorder` | 录制流程：采集、切句、转写、入库 |
| `internal/processor` | 会后流程：整段重新转写、区分说话人、按声音复核和认人、混音 |
| `internal/notes` | 纪要：会中大纲、会后定稿、白板节点 |
| `internal/audio` | 降采样、按语音概率切句 |
| `internal/asr`、`internal/llm` | 转写服务和模型服务的客户端 |
| `internal/store`、`internal/config`、`internal/export` | SQLite 存储、设置文件、导出（逐字稿和纪要的 Markdown） |
| `internal/library`、`internal/control` | 跨会议搜索；命令行向运行中的应用提请求的 socket |
| `modules/capture` | 原生模块：麦克风和系统声音采集（macOS 用 Swift，Windows 用 Rust） |
| `modules/speech` | 原生模块（Rust）：语音检测、录音编解码、导入音视频、说话人识别 |
| `resources/models` | 说话人识别用的三个 ONNX 模型 |
| `src` | 界面 |

## 录制

每个来源一条轨：麦克风是「我」，电脑里的声音是「对方」。采集 48kHz 单声道，边录边编码成 Ogg Opus 落盘（每条轨 32kbps）。同时降到 16kHz 送 silero-vad，按停顿切成句子，逐句送去转写。

## 会后

停止录制后（或导入一个音视频文件后）在后台整理：每条轨整段重新送去转写，拿到按句子切分、带时间的定稿（一口气说下来、服务没切开的长段，按句末标点和逐字时间再切成句子）；多人说话的那条轨在本地做说话人识别；两条轨混成一个文件供回放。失败时保留会中转写的文字稿，可以重试。

自定义的转写服务只要求返回文字，不要求时间戳。录音在本地用 pyannote 找出语音（离麦克风远的人 silero-vad 听不到），在停顿和换人的地方切成最长 20 秒的段，8 段同时送去转写；每段的时间取切点，段里有几句就按字数分摊。热词作为 `prompt` 传给服务。

## 纪要和白板

会中每约 45 秒把新的发言交给模型，更新一份按话题排列的大纲：只往后追加，只有最后两个话题还会被改写，每条要点都带着它依据的那句话的编号，没有出处的内容会被丢掉。会后用完整的定稿文字稿另写一份纪要（标题、结论、待办、还没定的事、话题）；会议没被改过名时，标题换成纪要起的这一个。同一次调用还写出时间轴：按时间把会议分成首尾相接的几章，每章一个标题和一两句经过，点一章从那里播放，进度条上有章节刻度。给说话人改名或合并说话人时，纪要和时间轴里的旧称呼跟着换；待办的负责人是会上的说话人时显示头像，点开可以改名。

会后的纪要是一篇可编辑的文档（ProseKit）：时间戳、负责人、存疑是文档里的行内节点，点时间戳跳回原句；编辑后的文档单独保存，重新生成时保留上一版。

白板是 Excalidraw 画布，脑图由纪要里每个话题的短语节点生成，位置由程序排。用户移动过、改过、删过的节点不会再被自动改动，用户自己画的内容永远不动。

## 说话人识别

停止录制后在本地运行，回答「这场会里谁说了哪句」。实现在 `modules/speech`（Rust），macOS 和 Windows 共用同一份代码。整条流程移植自 [senko](https://github.com/narcotic-sh/senko)，它是一套算法，本身不是模型；流程里用到两个模型。

| 步骤 | 做什么 | 用什么 | 状态 |
| --- | --- | --- | --- |
| 1. 找出语音段 | 在整段录音里标出有人说话的时间段，去掉静音和噪声 | 模型：pyannote segmentation-3.0（ONNX，6 MB） | 已实现 |
| 2. 切小窗 | 把语音段切成 1.5 秒的小窗，每 0.6 秒滑一次 | 代码 | 已实现 |
| 3. 提取特征 | 每个小窗算出 80 维的 fbank 声学特征 | 代码（rustfft） | 已实现 |
| 4. 算声音向量 | 每个小窗变成一个 192 维向量，同一个人的向量彼此接近 | 模型：CAM++（ONNX，28 MB，用 `scripts/export_campplus.py` 导出） | 已实现 |
| 5. 聚类 | 把所有向量分成若干组，一组就是一个人，人数自动估计 | 代码：谱聚类（faer 做特征分解） | 已实现 |
| 6. 拼成时间段 | 相邻同组的小窗合并成「某人从几分几秒说到几分几秒」，按说话时长排序编号 | 代码 | 已实现 |
| 7. 句子归属 | 转写里的每句话，归给时间重叠最多的那个人；重叠不到六成的标「存疑」 | 代码（Go，`internal/processor`） | 已实现 |
| 8. 跨会议认人 | 给每个人算一个声纹，和本地声纹库比对，认出这是谁 | 模型：ReDimNet2 b6（ONNX，53 MB） | 已实现 |

模型都由 ONNX Runtime 运行（`ort` 2.0.0-rc.15）。第 8 步和逐句复核用第三个模型 ReDimNet2 b6：用户在说话人面板里打开「记住声音」后，这个人的声纹存进本地库，以后的会议里相似度到 0.52 的说话人自动用这个名字。

在 Kept 的 25 场金标上的句级归属准确率（自动估计人数），以及第 1 步换成 silero-vad 的对比：

| 第 1 步用的模型 | 中文 20 场 | 英文 5 场 |
|---|---|---|
| pyannote segmentation-3.0（现用） | 94.73% | 92.38% |
| silero-vad v6.2.3（阈值 0.55，未调参） | 93.69% | 90.28% |

silero-vad 用在别处：会中实时切句。

细节见 `docs/research/tech-validation-2026-11-21.md`。整体范围和阶段见 `docs/plan.md`。
