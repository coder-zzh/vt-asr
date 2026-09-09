#!/bin/bash
set -e

# 模型下载目录（可在环境变量中覆盖）
MODEL_DIR="${VT_MODEL_DIR:-./models/sense-voice}"
MODEL_URL="https://huggingface.co/k2-fsa/sherpa-onnx/resolve/main/sense-voice/model.int8.onnx"
TOKENS_URL="https://huggingface.co/k2-fsa/sherpa-onnx/resolve/main/sense-voice/tokens.txt"

echo "=== vt-asr 模型下载脚本 ==="
echo "下载目录: $MODEL_DIR"

mkdir -p "$MODEL_DIR"

if [ ! -f "$MODEL_DIR/model.int8.onnx" ]; then
    echo "下载模型文件..."
    wget -O "$MODEL_DIR/model.int8.onnx" "$MODEL_URL"
else
    echo "模型文件已存在"
fi

if [ ! -f "$MODEL_DIR/tokens.txt" ]; then
    echo "下载 tokens 文件..."
    wget -O "$MODEL_DIR/tokens.txt" "$TOKENS_URL"
else
    echo "tokens 文件已存在"
fi

echo ""
echo "=== 下载完成 ==="
echo "设置环境变量："
echo "  export VT_MODEL_PATH=$MODEL_DIR/model.int8.onnx"
echo "  export VT_TOKENS_PATH=$MODEL_DIR/tokens.txt"
