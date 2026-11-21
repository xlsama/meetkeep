# 技术验证：Windows 采集与 Rust 说话人分离（2026-11-21）

## 结论

两项都通过。MyGo 的 native module 在 Windows 上能加载 Rust 实现，按进程采集和本地说话人分离都可用。

## Windows 音频采集

实现在 `modules/capture/rust`（`wasapi` 0.27），直接向 WASAPI 请求 16 kHz 单声道 16 位，不自己重采样。

GitHub Actions `windows-latest`（装 VB-CABLE 虚拟声卡）上的结果，run 44873472049：

| 测试 | 结果 |
|---|---|
| 只录指定进程（含子进程） | 30 帧 / 3 秒，峰值 RMS 0.35 |
| 全部系统声音，排除自身进程树 | 30 帧，峰值 RMS 0.35 |
| 只录自身进程，同时别的进程在放音 | 30 帧，峰值 RMS 0.0000 |
| 麦克风 | 20 帧 / 2 秒，峰值 RMS 0.35 |
| 经 MyGo 从 Go 调用系统采集 | 29 帧，峰值 RMS 0.35 |

踩到的坑：

- 麦克风隐私开关关闭时 WASAPI 返回 `0x80070005`，已映射为 `capture.ErrDenied`。
- Windows 检出默认转 CRLF，MyGo 认不出 `//mygo:native`，仓库用 `.gitattributes` 统一 LF。
- `wasapi` 的 `new_application_loopback_client(pid, false)` 实际是「排除该进程树」，文档注释写反了。
- 进程回环在没人出声时不投递数据，实现里按等待超时补静音帧。

没有验证的：真实硬件上的麦克风、回声消除（Windows 端还没做）、列出可选 App 的接口、Windows ARM64。

## Rust 说话人分离

实现在 `modules/diarize/rust`，是 Kept `SpectralSpeakerEngine` 的移植：pyannote segmentation 3.0 出语音区域 → 1.5 s / 0.6 s 滑窗 → CAM++ 向量 → 无阈值谱聚类。ONNX Runtime 1.31（`ort` 2.0.0-rc.15），特征分解用 `faer`，两端同一份代码。

用 Kept 的 25 场金标和同一个打分器（`eval/diarization/run.py::score`，句级归属，自动估人数）：

| | Rust 模块 | Kept 现役 |
|---|---|---|
| 中文 AISHELL-4 20 场 | 94.73%（7714/8143） | 94.90%（7728/8143） |
| 英文 AMI 5 场 | 92.38%（703/761） | 92.25%（702/761） |
| 中文人数完全正确 | 18/20 | 17/20 |

- 39 分钟会议在 M 系列 Mac 上约 25 秒（含进程启动）。
- Windows 与 macOS 对同一段四人样例输出完全相同的分段。

### 向量模型必须自己导出

sherpa-onnx 发布的 `3dspeaker_speech_campplus_sv_zh_en_16k-common_advanced.onnx` 与 Kept 的 CoreML 版不是同一份权重：同一输入余弦只有 0.73，用它中文只有 92.94%、人数正确 15/20。

`scripts/export_campplus.py` 从 senko 仓库里的原始权重导出 ONNX，与 CoreML 版余弦 0.9998，上表用的是这份。正式项目需要自己托管这个文件（约 28 MB）。

### 语音段检测能不能换成 silero-vad

不能直接换。把第一步的 pyannote segmentation-3.0 换成 silero-vad v6.2.3（senko 的参数：阈值 0.55、最短语音 250 ms、最短静音 100 ms），其余不变：

| | pyannote | silero |
|---|---|---|
| 中文 20 场 | 94.73% | 93.69% |
| 英文 5 场 | 92.38% | 90.28% |

英文掉得多是因为 ami-TS3003a 一场 4 人只认出 2 人（82.5%）。silero 的阈值没有调过。换掉只省 6 MB，所以保留 pyannote；试验代码没有留在模块里。

### MyGo 的一个构建问题

`mygo native build` 用 `--release` 加 `MACOSX_DEPLOYMENT_TARGET` 编译时，过程宏动态库被 strip 后加载失败，报 `can't find crate`。`Cargo.toml` 里加了：

```toml
[profile.release.build-override]
strip = false
```

## 顺带测到的：MAI-Transcribe-2 说话人分离的时长限制

Azure 直连，一场中文 6 人会议：10 分钟 93.8%、20 分钟 86.0%，两个 38 分钟文件都在 10 分钟内没有返回。
