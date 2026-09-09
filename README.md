# vt-asr

> Offline speech-to-text server for Linux desktops.
> Go + sherpa-onnx + SenseVoice，专为中文优化的离线语音识别服务。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](#license)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](#requirements)
[![Platform](https://img.shields.io/badge/Platform-Linux-016FE0.svg)](#supported-platforms)

---

## Features

- **Go 实现** — 市面上唯一的 Go + CGo ASR server
- **内存优化 58%** — 382MB vs Python 版 928MB
- **中文优化** — SenseVoice 模型，比 Whisper 中文好
- **生产级稳定性** — PID 锁、watchdog、内存重建
- **单二进制** — 3.1MB，无需运行时
- **systemd 集成** — 开机自启、资源限制

## Quick Start

### 1. 克隆仓库

```bash
git clone https://github.com/coder-zzh/vt-asr.git
cd vt-asr
```

### 2. 下载模型

```bash
bash setup.sh
```

### 3. 构建并运行

```bash
make build
export VT_MODEL_PATH=./models/sense-voice/model.int8.onnx
export VT_TOKENS_PATH=./models/sense-voice/tokens.txt
./vt-asr-server
```

### 4. 测试识别

```bash
echo "/path/to/test.wav" | nc -U /tmp/vt_asr.sock
```

## Architecture

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  麦克风   │───▶│ PipeWire │───▶│  Go ASR  │───▶│  文字输出 │
│ (耳麦/内置)│    │ (AEC)    │    │ (CGo)    │    │ (剪贴板) │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                      │               │
                      ▼               ▼
                 WAV 文件      Unix Socket
                                /tmp/vt_asr.sock
```

## Installation

### 从源码构建

**系统要求：**
- Linux（Wayland 或 X11）
- Go 1.23+
- PipeWire（用于 AEC 回声消除）
- sherpa-onnx（通过 pip 安装）

**步骤：**

```bash
# 安装 Go（如果未安装）
wget https://go.dev/dl/go1.23.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 安装 sherpa-onnx
pip install sherpa-onnx

# 下载模型
bash setup.sh

# 构建
make build
```

### 环境变量配置

| 变量 | 默认值 | 说明 |
|---|---|---|
| `VT_MODEL_PATH` | （必填） | SenseVoice 模型路径 |
| `VT_TOKENS_PATH` | （必填） | Tokens 文件路径 |
| `VT_LISTEN` | `/tmp/vt_asr.sock` | Unix socket 监听地址 |
| `VT_PID_FILE` | `/tmp/vt_asr.pid` | PID 文件路径 |
| `VT_LOG` | `/tmp/vt_asr_server.log` | 日志文件路径 |
| `VT_MAX_MEMORY_MB` | `1536` | 内存限制 (MB) |
| `VT_REBUILD_EVERY` | `500` | 每 N 次请求重建 recognizer |
| `SHERPA_ONNX_DIR` | （必填） | sherpa-onnx 安装目录 |

### systemd 服务安装

```bash
# 复制服务文件（根据你的环境修改路径）
cp vt-asr.service ~/.config/systemd/user/

# 启动服务
systemctl --user daemon-reload
systemctl --user start vt-asr
systemctl --user enable vt-asr

# 查看状态
systemctl --user status vt-asr
```

## Usage

### 命令行测试

```bash
# 识别单个文件
echo "/path/to/audio.wav" | nc -U /tmp/vt_asr.sock

# 示例输出
开饭时间早上9点至下午5点。
```

### Python 客户端

```python
import socket

def recognize(wav_path):
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    sock.connect("/tmp/vt_asr.sock")
    sock.send(wav_path.encode())
    result = sock.recv(4096).decode()
    sock.close()
    return result

text = recognize("/path/to/audio.wav")
print(text)
```

## Performance

| 指标 | 值 |
|---|---|
| 模型大小 | 228MB (int8) |
| 内存占用 | 382MB RSS |
| 识别速度 | 0.1-0.3s/请求 |
| 吞吐量 | ~10 req/s |
| CPU 占用 | 4 线程 |

### 与 Python 版对比

| 指标 | Go 版 | Python 版 | 提升 |
|---|---|---|---|
| 内存 | 382MB | 928MB | -58% |
| 速度 | 0.1s | 0.08s | -25% |
| 二进制大小 | 3.1MB | N/A | - |
| 依赖 | 无 | Python venv | - |

## Supported Platforms

| 平台 | 状态 | 备注 |
|---|---|---|
| Linux Wayland | ✅ | 推荐 |
| Linux X11 | ⚠️ | 需要 xdotool |
| macOS | ❌ | 未测试 |
| Windows | ❌ | 不支持 |

## Troubleshooting

### 常见问题

**Q: 启动失败 "创建 recognizer 失败"**
A: 检查模型路径是否正确，运行 `bash setup.sh` 下载模型

**Q: 内存超限退出**
A: watchdog 每 30s 检查，超 1.5GB 自动退出。检查是否有内存泄漏

**Q: 识别结果为空**
A: 检查 WAV 文件格式（16kHz, 16bit, mono）

**Q: 无法连接 socket**
A: 检查服务是否运行：`systemctl --user status vt-asr`

## Contributing

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解开发规范。

## License

[MIT License](LICENSE) - Copyright (c) 2026 coder-zzh

---

**English Summary:**

vt-asr is an offline speech-to-text server for Linux desktops, built with Go + sherpa-onnx + SenseVoice model. It provides 58% memory reduction compared to Python alternatives, with production-grade stability features including PID lock, memory watchdog, and periodic model rebuild.

Key features:
- Go + CGo implementation (unique in the market)
- Chinese-optimized SenseVoice model
- 382MB memory usage (vs 928MB Python)
- Single 3.1MB binary
- systemd integration with resource limits
- Unix socket server for easy integration
