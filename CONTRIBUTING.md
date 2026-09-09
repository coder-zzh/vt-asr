# 贡献指南

感谢你对 vt-asr 的关注！

## 开发环境

### 系统要求
- Linux（推荐 Ubuntu 22.04+）
- Go 1.23+
- PipeWire

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/coder-zzh/vt-asr.git
cd vt-asr

# 安装依赖
pip install sherpa-onnx

# 下载模型
bash setup.sh

# 构建
make build

# 运行测试
make test
```

## 代码规范

### Go 代码
- 遵循 [Effective Go](https://go.dev/doc/effective_go) 规范
- 使用 `gofmt` 格式化
- 函数注释使用中文

### Commit 规范
- 使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式
- 示例：`feat: 添加 GPU 加速支持`、`fix: 修复内存泄漏`

### PR 流程
1. Fork 仓库
2. 创建特性分支：`git checkout -b feat/my-feature`
3. 提交更改：`git commit -m 'feat: 添加新功能'`
4. 推送分支：`git push origin feat/my-feature`
5. 创建 Pull Request

## 报告问题

使用 [GitHub Issues](https://github.com/coder-zzh/vt-asr/issues) 报告问题，请包含：
- 系统信息（`uname -a`）
- 错误日志
- 复现步骤

## 许可证

贡献即表示你同意你的代码在 MIT 许可证下发布。
