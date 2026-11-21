# 说话人识别用的模型

| 文件 | 模型 | 用途 | 来源 | 许可 |
|---|---|---|---|---|
| `segmentation.onnx` | pyannote segmentation-3.0 | 标出语音段 | [sherpa-onnx 的发布包](https://github.com/k2-fsa/sherpa-onnx/releases/tag/speaker-segmentation-models) | MIT |
| `embedding.onnx` | CAM++（3D-Speaker campplus，192 维） | 会议内分人 | 用 `scripts/export_campplus.py` 从 [senko](https://github.com/narcotic-sh/senko) 的权重导出 | Apache-2.0 |
| `recognition.onnx` | ReDimNet2 b6（vb2+vox2+cnc2，large margin，192 维） | 逐句复核、跨会议认人 | 从官方 PyTorch 权重用 `torch.onnx.export` 导出，输入是 16kHz 波形，梅尔前端在图内 | MIT |
