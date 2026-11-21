# MeetKeep

开源的桌面会议记录应用：录制会议，实时转写，生成纪要和白板。macOS 和 Windows 双端，数据存本地。用 [MyGo](https://github.com/egoist/mygo) 写成：界面是 React，业务是 Go，采集和语音模型是 Swift 和 Rust 的原生模块。

范围和阶段见 `docs/plan.md`。

```sh
bun install
bun run dev
bun run check && bun run test
```

需要 macOS 14.2+、Go 1.27+、Bun、Xcode、Rust。Windows 上的采集和语音模块在 CI 里验证，应用界面还没在真机上跑过。

转写用 Azure 的 MAI-Transcribe-2，纪要和白板用任意 OpenAI 兼容的模型，都在应用的设置页里填。设置存在 `~/.config/meetkeep/settings.json`，会议存在同目录的 `meetkeep.db`，录音默认存在 `~/Documents/MeetKeep`。

## 结构

| 位置 | 内容 |
|---|---|
| `main.go`、`windows.go`、`services.go`、`events.go` | 应用入口、窗口和菜单栏、给界面调用的服务和事件 |
| `internal/recorder` | 录制流程：采集、切句、转写、入库 |
| `internal/audio` | 降采样、按语音概率切句 |
| `internal/asr`、`internal/llm` | 转写服务和模型服务的客户端 |
| `internal/store`、`internal/config` | SQLite 存储、设置文件 |
| `modules/capture` | 原生模块：麦克风和系统声音采集（macOS 用 Swift，Windows 用 Rust） |
| `modules/speech` | 原生模块（Rust）：语音检测、录音编码、说话人识别 |
| `src` | 界面 |

## 录制

每个来源一条轨：麦克风是「我」，电脑里的声音是「对方」。采集 48kHz 单声道，边录边编码成 Ogg Opus 落盘（每条轨 32kbps）。同时降到 16kHz 送 silero-vad，按停顿切成句子，逐句送去转写。

## 说话人识别

停止录制后在本地运行，回答「这场会里谁说了哪句」。实现在 `modules/speech`（Rust），macOS 和 Windows 共用同一份代码。整条流程移植自 [senko](https://github.com/narcotic-sh/senko)，它是一套算法，本身不是模型；流程里用到两个模型。

| 步骤 | 做什么 | 用什么 | 状态 |
|---|---|---|---|
| 1. 找出语音段 | 在整段录音里标出有人说话的时间段，去掉静音和噪声 | 模型：pyannote segmentation-3.0（ONNX，6 MB） | 已实现 |
| 2. 切小窗 | 把语音段切成 1.5 秒的小窗，每 0.6 秒滑一次 | 代码 | 已实现 |
| 3. 提取特征 | 每个小窗算出 80 维的 fbank 声学特征 | 代码（rustfft） | 已实现 |
| 4. 算声音向量 | 每个小窗变成一个 192 维向量，同一个人的向量彼此接近 | 模型：CAM++（ONNX，28 MB，用 `scripts/export_campplus.py` 导出） | 已实现 |
| 5. 聚类 | 把所有向量分成若干组，一组就是一个人，人数自动估计 | 代码：谱聚类（faer 做特征分解） | 已实现 |
| 6. 拼成时间段 | 相邻同组的小窗合并成「某人从几分几秒说到几分几秒」，按说话时长排序编号 | 代码 | 已实现 |
| 7. 句子归属 | 转写里的每句话，归给时间重叠最多的那个人 | 代码 | 未实现 |
| 8. 跨会议认人 | 给每个人算一个声纹，和本地声纹库比对，认出这是谁 | 模型：专门的声纹模型（Kept 用 ReDimNet2 b6，53 MB） | 未实现，第二版 |

两个模型都由 ONNX Runtime 运行（`ort` 2.0.0-rc.15）。

在 Kept 的 25 场金标上的句级归属准确率（自动估计人数），以及第 1 步换成 silero-vad 的对比：

| 第 1 步用的模型 | 中文 20 场 | 英文 5 场 |
|---|---|---|
| pyannote segmentation-3.0（现用） | 94.73% | 92.38% |
| silero-vad v6.2.3（阈值 0.55，未调参） | 93.69% | 90.28% |

silero-vad 用在别处：会中实时切句。

细节见 `docs/research/tech-validation-2026-11-21.md`。整体范围和阶段见 `docs/plan.md`。
