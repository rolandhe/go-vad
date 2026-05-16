# go-vad

[![Go Reference](https://pkg.go.dev/badge/github.com/rolandhe/go-vad.svg)](https://pkg.go.dev/github.com/rolandhe/go-vad)

纯 Go 语音活动检测 (VAD) 库 —— [libfvad](https://github.com/dpirch/libfvad) 的忠实移植，libfvad 是从 [WebRTC](https://webrtc.org/) 原生代码包中提取的独立 VAD 引擎。

**零外部依赖。无 cgo。** 在所有采样率、模式和帧长度下，输出与 C 参考实现逐位一致。

## 安装

```bash
go get github.com/rolandhe/go-vad
```

## 快速开始

```go
package main

import (
    "fmt"
    "github.com/rolandhe/go-vad"
)

func main() {
    // 创建新的 VAD 实例。
    v := vad.New()

    // 配置：16 kHz 采样率，激进模式。
    v.SetSampleRate(vad.SampleRate16k)
    v.SetMode(vad.ModeAggressive)

    // 处理 20ms 帧（16 kHz 下为 320 个采样点）。
    frame := make([]int16, 320)
    // ... 用 16 位 PCM 音频数据填充帧 ...

    result, err := v.Process(frame)
    if err != nil {
        panic(err)
    }
    if result == vad.ResultVoice {
        fmt.Println("检测到语音")
    } else {
        fmt.Println("静音 / 非语音")
    }
}
```

## API 参考

### 类型

```go
type Mode int
```

VAD 激进程度模式。更高的模式更严格 —— 它们将更少的帧分类为语音，减少误报但增加漏检。

| 常量 | 值 | 描述 |
|----------|-------|-------------|
| `ModeQuality` | `0` | 普通质量（默认） |
| `ModeLowBitrate` | `1` | 低比特率 |
| `ModeAggressive` | `2` | 激进 |
| `ModeVeryAggressive` | `3` | 非常激进 |

```go
type SampleRate int
```

有效的输入采样率，单位为 Hz。

| 常量 | 值 |
|----------|-------|
| `SampleRate8k` | `8000` |
| `SampleRate16k` | `16000` |
| `SampleRate32k` | `32000` |
| `SampleRate48k` | `48000` |

```go
type Result int
```

每帧的 VAD 判定结果。

| 常量 | 值 | 描述 |
|----------|-------|-------------|
| `ResultNoVoice` | `0` | 未检测到活动语音 |
| `ResultVoice` | `1` | 检测到活动语音 |

### 构造函数

```go
func New() *VAD
```

创建并初始化一个新的 VAD 实例，默认设置为：
- 模式：`ModeQuality` (0)
- 采样率：`SampleRate8k` (8000 Hz)

### 方法

```go
func (v *VAD) SetMode(mode Mode) error
```

设置激进程度模式。如果 mode 不在 0–3 范围内则返回错误。

```go
func (v *VAD) SetSampleRate(sr SampleRate) error
```

设置输入采样率。有效值：`8000`、`16000`、`32000`、`48000`。无效采样率将返回错误。内部所有处理均在 8 kHz 下进行 —— 更高的采样率会先降采样。

```go
func (v *VAD) Process(frame []int16) (Result, error)
```

处理一帧 16 位有符号 PCM 音频并返回 VAD 判定结果。

帧长度必须对应配置采样率下的 **10 ms、20 ms 或 30 ms**：

| 采样率 | 10 ms | 20 ms | 30 ms |
|-------------|-------|-------|-------|
| 8000 Hz | 80 | 160 | 240 |
| 16000 Hz | 160 | 320 | 480 |
| 32000 Hz | 320 | 640 | 960 |
| 48000 Hz | 480 | 960 | 1440 |

帧长度无效时返回错误。

```go
func (v *VAD) Reset()
```

重新初始化 VAD 实例，清除所有内部状态并将模式和采样率重置为默认值。

## 帧大小参考

根据应用需求选择合适的帧时长：

| 时长 | 延迟 | 灵敏度 |
|----------|---------|-------------|
| 10 ms | 最低 | 适合短语音 |
| 20 ms | 中等 | 均衡 |
| 30 ms | 较高 | 最佳频率分辨率 |

## 并发

每个 `*VAD` 实例是**有状态的**，不支持多个 goroutine 并发使用。如果需要并发处理多个音频流，请为每个流创建单独的 `*VAD` 实例。

库本身没有全局可变状态，并发实例化是安全的。

## 算法概述

VAD 在 8 kHz 音频上运行，工作流程如下：

1. **降采样**：将 >8 kHz 的输入通过基于全通滤波器的抽取器降采样至 8 kHz。
2. **特征提取**：正交镜像滤波器 (QMF) 组将 0–4 kHz 频谱划分为 6 个子带（80–250、250–500、500–1000、1000–2000、2000–3000、3000–4000 Hz）。每个子带的 log 能量构成一个 6 维特征向量。
3. **GMM 评分**：高斯混合模型（2 个高斯 × 6 个通道）计算特征向量在语音和噪声假设下的似然度。
4. **决策**：根据模式相关的阈值进行局部（各子带）和全局对数似然比检验。GMM 参数会根据输入信号进行在线自适应。
5. **拖尾平滑**：当语音结束时，VAD 会在短暂时间内继续报告语音，以避免词语中途截断。

所有运算均使用**定点**（Q 格式）算术，与 WebRTC 参考实现完全一致。

## 许可证

BSD 3-Clause。详见 [LICENSE](LICENSE) 文件。基于 WebRTC 代码（版权归属 The WebRTC project authors）和 libfvad（版权归属 Daniel Pirch）。
