# meetkeep-poc

MyGo 0.10 上的会议转写 POC：Swift native module 采麦克风和系统音频，Go 按停顿切句后调 Azure MAI-Transcribe-2，React 页面实时显示。

```sh
bun install
cp .env.example .env   # 填 Azure Speech 的 key，以及任意 OpenAI 兼容 LLM 的地址、密钥、模型名
bun run dev
bun run check && bun run test
```

需要 macOS 14.2+、Go 1.27+、Bun、Xcode、Rust。Windows 上的采集和说话人识别模块已在 CI 里验证，应用界面还没在真机上跑过。

## 说话人识别

停止录制后在本地运行，回答「这场会里谁说了哪句」。实现在 `modules/diarize`（Rust，约 700 行），macOS 和 Windows 共用同一份代码。整条流程移植自 [senko](https://github.com/narcotic-sh/senko)，它是一套算法，本身不是模型；流程里用到两个模型。

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

silero-vad 仍然要用，但用在别处：会中实时切句，还没接入。

细节见 `docs/research/tech-validation-2026-11-21.md`。整体范围和阶段见 `docs/plan.md`。
