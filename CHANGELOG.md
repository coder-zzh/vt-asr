# Changelog

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [0.1.0] - 2026-09-09

### 新增
- Go 实现的 ASR server
- SenseVoice 模型支持
- Unix socket 通信
- PID 锁防止重复实例
- 内存 watchdog（1.5GB 限制）
- 每 500 次请求重建 recognizer
- systemd 服务集成
- 双语 README（中文 + 英文）

### 性能
- 内存优化 58%（382MB vs Python 版 928MB）
- 识别速度 0.1-0.3s/请求
