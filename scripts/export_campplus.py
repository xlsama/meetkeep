"""Exports CAM++ (3D-Speaker speech_campplus_sv_zh_en_16k-common_advanced) to ONNX
from the weights and model class in a checkout of https://github.com/narcotic-sh/senko.

    uv run --python 3.12 --with torch,onnx python3 -I scripts/export_campplus.py <senko checkout> embedding.onnx
"""

import importlib.util
import sys

import torch

senko, out = sys.argv[1:3]
spec = importlib.util.spec_from_file_location("camplusplus", f"{senko}/senko/camplusplus.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

model = module.CAMPPlus(feat_dim=80, embedding_size=192, memory_efficient=False)
weights = f"{senko}/models/speech_campplus_sv_zh_en_16k-common_advanced/campplus_cn_en_common.pt"
model.load_state_dict(torch.load(weights, map_location="cpu", weights_only=True))
model.eval()

torch.onnx.export(
    model,
    (torch.randn(2, 150, 80),),
    out,
    input_names=["x"],
    output_names=["embedding"],
    dynamic_axes={"x": {0: "N"}, "embedding": {0: "N"}},
    opset_version=17,
    dynamo=False,
)
