# 说话人识别用的模型

| 文件 | 模型 | 来源 | 许可 |
|---|---|---|---|
| `segmentation.onnx` | pyannote segmentation-3.0 | [sherpa-onnx 的发布包](https://github.com/k2-fsa/sherpa-onnx/releases/tag/speaker-segmentation-models) | MIT |
| `embedding.onnx` | CAM++（3D-Speaker campplus，192 维） | 用 `scripts/export_campplus.py` 从 [senko](https://github.com/narcotic-sh/senko) 的权重导出 | Apache-2.0 |
