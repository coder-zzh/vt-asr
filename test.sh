#!/bin/bash
set -e

SOCK="/tmp/vt_asr.sock"
SERVER="./vt-asr-server"
LOG="/tmp/vt_asr_server.log"

cleanup() {
    echo "[test] 清理..."
    kill $PID 2>/dev/null; wait $PID 2>/dev/null
    rm -f "$SOCK"
}
trap cleanup EXIT

echo "[test] 启动 Go ASR server..."
rm -f "$SOCK" "$LOG"

# 检查环境变量
if [ -z "$VT_MODEL_PATH" ] || [ -z "$VT_TOKENS_PATH" ]; then
    echo "[test] 错误: 请设置 VT_MODEL_PATH 和 VT_TOKENS_PATH 环境变量"
    exit 1
fi

"$SERVER" > /dev/null 2>&1 &
PID=$!
echo "[test] PID=$PID"

# 等待 socket 就绪
for i in $(seq 1 10); do
    if [ -S "$SOCK" ]; then
        echo "[test] Socket 就绪 (${i}00ms)"
        break
    fi
    sleep 0.5
done

if [ ! -S "$SOCK" ]; then
    echo "[test] ERROR: Socket 超时"
    tail -5 "$LOG"
    exit 1
fi

# 内存
echo "[test] 模型加载后内存:"
cat /proc/$PID/status | grep -E "^(VmRSS|VmSize|VmSwap)"

echo "[test] 测试通过!"
