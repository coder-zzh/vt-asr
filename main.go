package main

/*
#cgo CFLAGS: -I/usr/include/sherpa-onnx
#cgo LDFLAGS: -L/usr/lib -lsherpa-onnx-c-api -lonnxruntime
#include <stdlib.h>
#include <string.h>
#include "sherpa-onnx/c-api/c-api.h"

static void zero_config(SherpaOnnxOfflineRecognizerConfig *c) {
	memset(c, 0, sizeof(*c));
}
*/
import "C"

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// 配置：支持环境变量，默认值留空让用户必须设置
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	// 模型路径（必须设置 VT_MODEL_PATH 和 VT_TOKENS_PATH）
	modelPath  = getEnv("VT_MODEL_PATH", "")
	tokensPath = getEnv("VT_TOKENS_PATH", "")

	// 监听地址
	listenAddr = getEnv("VT_LISTEN", "/tmp/vt_asr.sock")

	// PID 文件
	pidFile = getEnv("VT_PID_FILE", "/tmp/vt_asr.pid")

	// 内存限制 (MB)
	maxMemoryMB = func() int {
		v := getEnv("VT_MAX_MEMORY_MB", "1536")
		n, _ := strconv.Atoi(v)
		if n <= 0 {
			return 1536
		}
		return n
	}()

	// 每 N 次请求重建 recognizer（防内存泄漏）
	rebuildEvery = func() int {
		v := getEnv("VT_REBUILD_EVERY", "500")
		n, _ := strconv.Atoi(v)
		if n <= 0 {
			return 500
		}
		return n
	}()
)

var (
	recognizer *C.SherpaOnnxOfflineRecognizer
	logger     *log.Logger
	logFile    *os.File
	reqCount   int64
	mu         sync.Mutex
)

func initLog() {
	logPath := getEnv("VT_LOG", "/tmp/vt_asr_server.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open log: %v", err)
	}
	logFile = f
	logger = log.New(f, "", 0)
}

func dbg(args ...interface{}) {
	prefix := time.Now().Format("15:04:05") + " "
	msg := fmt.Sprint(args...)
	logger.Println(prefix + msg)
}

func cleanup() {
	os.Remove(listenAddr)
	os.Remove(pidFile)
	if recognizer != nil {
		C.SherpaOnnxDestroyOfflineRecognizer(recognizer)
		recognizer = nil
	}
	if logFile != nil {
		logFile.Close()
	}
}

func killOldInstance() {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	oldPid := strings.TrimSpace(string(data))
	if oldPid == "" {
		return
	}
	pid, err := strconv.Atoi(oldPid)
	if err != nil {
		return
	}
	err = syscall.Kill(pid, 0)
	if err != nil {
		return
	}
	dbg("发现旧进程 PID=", pid, "，发送 SIGTERM")
	syscall.Kill(pid, syscall.SIGTERM)
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		err = syscall.Kill(pid, 0)
		if err != nil {
			dbg("旧进程已退出")
			return
		}
	}
	dbg("旧进程未响应，发送 SIGKILL")
	syscall.Kill(pid, syscall.SIGKILL)
	time.Sleep(500 * time.Millisecond)
}

func writePidFile() {
	os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func watchdog() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		f, err := os.Open("/proc/self/status")
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, _ := strconv.Atoi(fields[1])
					mb := kb / 1024
					if mb > maxMemoryMB {
						dbg(fmt.Sprintf("watchdog: 内存 %dMB 超限 %dMB，退出", mb, maxMemoryMB))
						cleanup()
						os.Exit(1)
					}
				}
				break
			}
		}
		f.Close()
	}
}

func loadModel() {
	t0 := time.Now()
	dbg("加载 sense-voice:", modelPath)

	cModel := C.CString(modelPath)
	cTokens := C.CString(tokensPath)
	cLang := C.CString("auto")

	var config C.SherpaOnnxOfflineRecognizerConfig
	C.zero_config(&config)

	config.feat_config.sample_rate = 16000
	config.feat_config.feature_dim = 80

	config.model_config.sense_voice.model = cModel
	config.model_config.sense_voice.language = cLang
	config.model_config.sense_voice.use_itn = 1
	config.model_config.tokens = cTokens
	config.model_config.num_threads = 4
	config.model_config.provider = C.CString("cpu")
	config.decoding_method = C.CString("greedy_search")

	recognizer = C.SherpaOnnxCreateOfflineRecognizer(&config)

	C.free(unsafe.Pointer(cModel))
	C.free(unsafe.Pointer(cTokens))
	C.free(unsafe.Pointer(cLang))
	C.free(unsafe.Pointer(config.model_config.provider))
	C.free(unsafe.Pointer(config.decoding_method))

	if recognizer == nil {
		log.Fatalf("创建 recognizer 失败")
	}

	dbg(fmt.Sprintf("模型加载完成 %.1fs backend=sense-voice", time.Since(t0).Seconds()))
}

func rebuildModel() {
	mu.Lock()
	defer mu.Unlock()

	t0 := time.Now()
	dbg("watchdog: 重建 recognizer 防内存泄漏")

	C.SherpaOnnxDestroyOfflineRecognizer(recognizer)
	recognizer = nil

	cModel := C.CString(modelPath)
	cTokens := C.CString(tokensPath)
	cLang := C.CString("auto")

	var config C.SherpaOnnxOfflineRecognizerConfig
	C.zero_config(&config)

	config.feat_config.sample_rate = 16000
	config.feat_config.feature_dim = 80

	config.model_config.sense_voice.model = cModel
	config.model_config.sense_voice.language = cLang
	config.model_config.sense_voice.use_itn = 1
	config.model_config.tokens = cTokens
	config.model_config.num_threads = 4
	config.model_config.provider = C.CString("cpu")
	config.decoding_method = C.CString("greedy_search")

	recognizer = C.SherpaOnnxCreateOfflineRecognizer(&config)

	C.free(unsafe.Pointer(cModel))
	C.free(unsafe.Pointer(cTokens))
	C.free(unsafe.Pointer(cLang))
	C.free(unsafe.Pointer(config.model_config.provider))
	C.free(unsafe.Pointer(config.decoding_method))

	if recognizer == nil {
		log.Fatalf("重建 recognizer 失败")
	}

	reqCount = 0
	dbg(fmt.Sprintf("重建完成 %.1fs", time.Since(t0).Seconds()))
}

func readWavFloat32(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var riff [4]byte
	var fileSize uint32
	var wave [4]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, fmt.Errorf("读 RIFF: %w", err)
	}
	binary.Read(f, binary.LittleEndian, &fileSize)
	if _, err := io.ReadFull(f, wave[:]); err != nil {
		return nil, fmt.Errorf("读 WAVE: %w", err)
	}

	var chunkID [4]byte
	var chunkSize uint32
	for {
		if _, err := io.ReadFull(f, chunkID[:]); err != nil {
			return nil, fmt.Errorf("找 fmt: %w", err)
		}
		binary.Read(f, binary.LittleEndian, &chunkSize)
		if string(chunkID[:]) == "fmt " {
			break
		}
		f.Seek(int64(chunkSize), io.SeekCurrent)
	}

	var audioFormat uint16
	var numChannels uint16
	var sampleRate uint32
	var byteRate uint32
	var blockAlign uint16
	var bitsPerSample uint16
	binary.Read(f, binary.LittleEndian, &audioFormat)
	binary.Read(f, binary.LittleEndian, &numChannels)
	binary.Read(f, binary.LittleEndian, &sampleRate)
	binary.Read(f, binary.LittleEndian, &byteRate)
	binary.Read(f, binary.LittleEndian, &blockAlign)
	binary.Read(f, binary.LittleEndian, &bitsPerSample)

	if chunkSize > 16 {
		f.Seek(int64(chunkSize-16), io.SeekCurrent)
	}

	for {
		if _, err := io.ReadFull(f, chunkID[:]); err != nil {
			return nil, fmt.Errorf("找 data: %w", err)
		}
		binary.Read(f, binary.LittleEndian, &chunkSize)
		if string(chunkID[:]) == "data" {
			break
		}
		f.Seek(int64(chunkSize), io.SeekCurrent)
	}

	pcm := make([]byte, chunkSize)
	if _, err := io.ReadFull(f, pcm); err != nil {
		return nil, fmt.Errorf("读 data: %w", err)
	}

	n := len(pcm) / 2
	samples := make([]float32, n)
	for i := 0; i < n; i++ {
		s := int16(pcm[i*2]) | int16(pcm[i*2+1])<<8
		samples[i] = float32(s) / 32768.0
	}

	return samples, nil
}

func recognize(wavPath string) (string, error) {
	samples, err := readWavFloat32(wavPath)
	if err != nil {
		return "", err
	}

	mu.Lock()

	stream := C.SherpaOnnxCreateOfflineStream(recognizer)
	if stream == nil {
		mu.Unlock()
		return "", fmt.Errorf("创建 stream 失败")
	}

	cSamples := (*C.float)(unsafe.Pointer(&samples[0]))
	C.SherpaOnnxAcceptWaveformOffline(stream, 16000, cSamples, C.int32_t(len(samples)))
	C.SherpaOnnxDecodeOfflineStream(recognizer, stream)

	result := C.SherpaOnnxGetOfflineStreamResult(stream)
	text := ""
	if result != nil {
		text = C.GoString(result.text)
		C.SherpaOnnxDestroyOfflineRecognizerResult(result)
	}
	C.SherpaOnnxDestroyOfflineStream(stream)

	reqCount++
	needRebuild := reqCount >= int64(rebuildEvery)

	mu.Unlock()

	if needRebuild {
		rebuildModel()
	}

	return strings.TrimSpace(text), nil
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		dbg("读取失败:", err)
		return
	}

	req := strings.TrimSpace(string(buf[:n]))
	if req == "" {
		return
	}

	lines := strings.SplitN(req, "\n", 2)
	wavPath := lines[len(lines)-1]
	wavPath = strings.TrimSpace(wavPath)

	if !filepath.IsAbs(wavPath) {
		dbg("路径不是绝对路径:", wavPath)
		return
	}

	t0 := time.Now()
	text, err := recognize(wavPath)
	if err != nil {
		dbg("识别异常:", err)
		return
	}

	dbg(fmt.Sprintf("识别 %s -> %.2fs: %s", filepath.Base(wavPath), time.Since(t0).Seconds(), text))

	_, err = conn.Write([]byte(text))
	if err != nil {
		dbg("发送失败:", err)
	}
}

func main() {
	// 检查必需的环境变量
	if modelPath == "" || tokensPath == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须设置 VT_MODEL_PATH 和 VT_TOKENS_PATH 环境变量")
		fmt.Fprintln(os.Stderr, "示例:")
		fmt.Fprintln(os.Stderr, "  export VT_MODEL_PATH=/path/to/model.int8.onnx")
		fmt.Fprintln(os.Stderr, "  export VT_TOKENS_PATH=/path/to/tokens.txt")
		os.Exit(1)
	}

	initLog()

	killOldInstance()
	os.Remove(listenAddr)
	writePidFile()

	loadModel()
	go watchdog()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		dbg("收到信号", sig, "退出")
		cleanup()
		os.Exit(0)
	}()

	ln, err := net.Listen("unix", listenAddr)
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}
	defer ln.Close()
	os.Chmod(listenAddr, 0666)

	dbg("监听", listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			dbg("accept 异常:", err)
			time.Sleep(time.Second)
			continue
		}
		go handleConn(conn)
	}
}
