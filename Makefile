.PHONY: build test clean install uninstall run log fmt vet

# sherpa-onnx 安装目录（必须设置）
SHERPA_ONNX_DIR ?= $(error SHERPA_ONNX_DIR 未设置，请运行: export SHERPA_ONNX_DIR=/path/to/sherpa-onnx)

# CGo 编译参数
CGO_CFLAGS = -I$(SHERPA_ONNX_DIR)/include
CGO_LDFLAGS = -L$(SHERPA_ONNX_DIR)/lib -lsherpa-onnx-c-api -lonnxruntime -Wl,-rpath,$(SHERPA_ONNX_DIR)/lib

# 默认目标
all: build

# 构建
build:
	CGO_ENABLED=1 \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	go build -o vt-asr-server .

# 测试
test:
	@echo "请确保已设置环境变量 VT_MODEL_PATH 和 VT_TOKENS_PATH"
	bash test.sh

# 清理
clean:
	rm -f vt-asr-server
	rm -f /tmp/vt_asr.sock /tmp/vt_asr.pid

# 安装（systemd 服务）
install: build
	mkdir -p ~/.config/systemd/user
	cp vt-asr.service ~/.config/systemd/user/
	systemctl --user daemon-reload
	@echo "服务已安装，运行: systemctl --user start vt-asr"

# 卸载
uninstall:
	systemctl --user stop vt-asr 2>/dev/null || true
	systemctl --user disable vt-asr 2>/dev/null || true
	rm -f ~/.config/systemd/user/vt-asr.service
	systemctl --user daemon-reload
	@echo "服务已卸载"

# 开发模式运行
run: build
	./vt-asr-server

# 查看日志
log:
	tail -f /tmp/vt_asr_server.log

# 格式化代码
fmt:
	gofmt -w main.go

# 静态检查
vet:
	go vet ./...
