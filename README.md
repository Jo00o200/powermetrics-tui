# 🚀 PowerMetrics TUI

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go)
![Platform](https://img.shields.io/badge/platform-macOS-000000?style=for-the-badge&logo=apple)
![License](https://img.shields.io/badge/license-MIT-green?style=for-the-badge)
![Status](https://img.shields.io/badge/status-active-success?style=for-the-badge)

**A beautiful, real-time system monitoring dashboard for macOS**

[Features](#-features) • [Installation](#-installation) • [Usage](#-usage) • [Screenshots](#-screenshots) • [Contributing](#-contributing)

</div>

---

## 🎯 Overview

PowerMetrics TUI transforms macOS's powerful `powermetrics` utility into an intuitive, interactive terminal dashboard. Monitor your system's performance, power consumption, and thermal state with a beautiful interface that makes complex metrics accessible to everyone.

Perfect for developers, power users, and anyone curious about their Mac's performance characteristics.

## ✨ Features

### 📊 Comprehensive Monitoring
- **Per-CPU Interrupt Breakdown**: See IPI, Timer, and Total interrupts for each individual CPU core with historical sparklines
- **Dynamic CPU Frequency Scaling**: Real-time frequency monitoring for each E-core and P-core with auto-scaling graphs
- **Power Analytics**: Monitor CPU, GPU, ANE (Neural Engine), and DRAM power consumption in real-time
- **Thermal Management**: View temperature sensors and thermal pressure states (Nominal, Fair, Serious, Critical)
- **Battery Intelligence**: Track charge levels, power draw, and battery health with usage history
- **Process Insights**:
  - Top processes by CPU usage with individual sparklines showing historical trends
  - **Recently Exited Processes**: Track processes that have terminated in the last 5 minutes
  - Shows max CPU%, average CPU%, peak memory usage, and how long ago the process exited
  - Perfect for monitoring build tools, scripts, and temporary processes
- **I/O Statistics**: Monitor network (in/out MB/s) and disk (read/write MB/s) activity with live graphs
- **Memory Usage**: Track RAM and swap utilization with pressure indicators

### 🎨 Beautiful Interface
- **10 Specialized Views**: Each metric category has its own optimized display
- **Real-time Visualizations**:
  - Live-updating bar charts that auto-scale to your hardware's capabilities
  - Sparkline graphs showing trends for the last 30-120 samples
  - Per-process CPU and memory history sparklines
  - Per-core frequency history visualization
- **Smart Color Coding**:
  - 🔴 Red: Critical/High usage (>80% CPU, >2000 interrupts/s, >50% power)
  - 🟡 Yellow: Moderate usage (50-80% CPU, 1000-2000 interrupts/s)
  - 🟢 Green: Normal usage (<50% CPU, <1000 interrupts/s)
  - 🔵 Blue: Efficiency cores, network input, memory usage
- **Responsive Design**: Automatically adjusts to terminal size, showing more processes on larger screens

### 🤝 User-Friendly
- **Help System**: Built-in descriptions for technical terms (toggle with 'h')
- **Quick Navigation**:
  - Number keys (1-9, 0) for instant view switching
  - Tab/Shift+Tab or Arrow keys for sequential navigation
- **Cross-Architecture**:
  - Apple Silicon: Distinguishes E-cores (Efficiency) and P-cores (Performance)
  - Intel Macs: Shows all CPU cores with appropriate frequency ranges
- **Auto-Detection**: Intelligently adapts to your Mac's capabilities and maximum frequencies

## 📥 Installation

### Prerequisites
- macOS (any version with `powermetrics` utility)
- Go 1.21 or later
- Terminal with UTF-8 support

### Quick Install

```bash
# Clone the repository
git clone https://github.com/sderosiaux/powermetrics-tui.git
cd powermetrics-tui

# Build the application
go build -o powermetrics-tui

# Make it globally accessible (optional)
sudo cp powermetrics-tui /usr/local/bin/
```

### Install from Source

```bash
go install github.com/sderosiaux/powermetrics-tui@latest
```

## 🚀 Usage

### Quick Start

```bash
# Authenticate sudo (required for powermetrics)
sudo -v

# Launch with all metrics
powermetrics-tui

# Or specify specific metrics
powermetrics-tui --samplers cpu_power,gpu_power,thermal
```

### Navigation

| Key | Action |
|-----|--------|
| `1-9, 0` | Jump directly to specific views |
| `Tab` | Cycle through views |
| `h` or `?` | Toggle help descriptions |
| `q` | Quit application |

### Available Views

1. **Interrupts** - CPU interrupt statistics with per-core IPI/Timer breakdown
2. **Power** - Power consumption metrics (CPU/GPU/ANE/DRAM)
3. **Frequency** - CPU/GPU clock speeds with historical sparklines
4. **Processes** - Top processes with CPU/memory history sparklines
5. **Network** - Network I/O statistics with throughput graphs
6. **Disk** - Disk I/O statistics with read/write monitoring
7. **Thermal** - Temperature and thermal pressure monitoring
8. **Battery** - Battery status, health, and charging metrics
9. **System** - Overall system metrics and resource usage
0. **Combined** - All metrics in one comprehensive view

### Command-Line Options

```bash
powermetrics-tui [options]

Options:
  --samplers    Comma-separated list of samplers (default: all)
                Options: interrupts, cpu_power, gpu_power, thermal, battery
  --interval    Sampling interval in milliseconds (default: 1000)
  --combined    Start in combined view mode
  --debug       Enable debug output
```

### Usage Scenarios

#### 🔥 Performance Troubleshooting
```bash
# Monitor CPU throttling under load
powermetrics-tui --samplers cpu_power,thermal,frequency

# Watch for thermal throttling during intensive tasks
# View shows: Thermal Pressure (Nominal → Fair → Serious → Critical)
# CPU frequencies will drop when thermal limits are reached
```

#### 🔋 Battery Optimization
```bash
# Track power consumption while on battery
powermetrics-tui --samplers battery,cpu_power,gpu_power

# Identify power-hungry processes
# Switch to Processes view (4) to see CPU% with historical trends
# High CPU sparklines (████████) indicate consistent high usage
```

#### 🎮 Gaming/Graphics Performance
```bash
# Monitor GPU and CPU performance during gaming
powermetrics-tui --samplers gpu_power,frequency,thermal

# GPU power spikes indicate graphics-intensive operations
# P-core frequencies show performance core utilization
# Thermal view reveals if throttling is affecting FPS
```

#### 💻 Development Workflow
```bash
# Monitor system impact during builds/compilation
powermetrics-tui --interval 500  # Faster sampling for quick changes

# Example during Xcode build:
# - E-cores: 2100-2400 MHz (background indexing)
# - P-cores: 3800-4200 MHz (active compilation)
# - Power: 15-25W CPU, 5-10W GPU
# - Processes: clang/swift showing high CPU% with rising sparklines
```

#### 🔍 System Debugging
```bash
# Investigate high interrupt rates (kernel issues)
powermetrics-tui

# Switch to Interrupts view (1)
# Look for:
# - IPI > 2000/s per CPU (red) indicates excessive inter-core communication
# - Timer > 1500/s (yellow) suggests timer coalescing issues
# - Uneven distribution across cores points to IRQ affinity problems
```

## 📸 Screenshots & Examples

### Interrupts View - Per-CPU Breakdown
```
CPU INTERRUPTS (System interrupts per second)

CPU0 (E):  IPI:   234/s  Timer:   890/s  Total:  1124/s  ██████░░░░  ▃▄▅▆▇▆▅▄▃▂
CPU1 (E):  IPI:   156/s  Timer:   823/s  Total:   979/s  █████░░░░░  ▂▃▄▅▄▃▂▁▂▃
CPU2 (E):  IPI:   189/s  Timer:   756/s  Total:   945/s  █████░░░░░  ▄▅▆▅▄▃▄▅▆▅
CPU3 (E):  IPI:   203/s  Timer:   801/s  Total:  1004/s  █████░░░░░  ▅▆▇▆▅▄▃▄▅▆
CPU4 (P):  IPI:  1823/s  Timer:  1234/s  Total:  3057/s  ████████░░  ▇█████▇▆▅▄  ⚠️
CPU5 (P):  IPI:  2156/s  Timer:  1456/s  Total:  3612/s  █████████░  ████████▇▆  🔴
CPU6 (P):  IPI:   567/s  Timer:   890/s  Total:  1457/s  ███████░░░  ▃▄▅▆▇▆▅▄▃▄
CPU7 (P):  IPI:   432/s  Timer:   823/s  Total:  1255/s  ██████░░░░  ▄▅▄▃▂▃▄▅▆▅

System Total: 12433 interrupts/s
```

### Power Consumption View
```
POWER CONSUMPTION (Energy usage - affects battery life)

CPU:     15234.5 mW  ████████████████████████░░░░  (68% of max)
  Processor power consumption

GPU:      4892.3 mW  ████████░░░░░░░░░░░░░░░░░░░░  (27% of max)
  Graphics processor power

ANE:       312.5 mW  █░░░░░░░░░░░░░░░░░░░░░░░░░░░  (3% of max)
  Apple Neural Engine - AI/ML accelerator

DRAM:     1823.1 mW  ███░░░░░░░░░░░░░░░░░░░░░░░░░  (10% of max)
  Memory power consumption

Total System: 22.3W
```

### CPU Frequency View with Sparklines
```
CPU FREQUENCY (Clock speeds in MHz)

E-Cores (Efficiency):
  Core 0:  2064 MHz  ██████████████░░░░░░  ▃▄▅▆▇▆▅▄▃▂▁▂▃▄
  Core 1:  1896 MHz  █████████████░░░░░░░  ▂▃▄▅▄▃▂▁▂▃▄▅▆▅
  Core 2:  2104 MHz  ███████████████░░░░░  ▄▅▆▅▄▃▄▅▆▅▄▃▂▃
  Core 3:  2248 MHz  ████████████████░░░░  ▅▆▇▆▅▄▃▄▅▆▇█▇▆

P-Cores (Performance):
  Core 4:  3824 MHz  █████████████████░░░  ▆▇████▇▆▅▄▅▆▇█
  Core 5:  4056 MHz  ██████████████████░░  ████████▇▆▅▆▇█
  Core 6:  2890 MHz  █████████████░░░░░░░  ▃▄▅▆▇▆▅▄▃▄▅▆▇▆
  Core 7:  3124 MHz  ██████████████░░░░░░  ▄▅▆▇▆▅▄▃▄▅▆▇█▇

GPU:      1398 MHz  ████████████░░░░░░░░
```

### Processes View with History Sparklines
```
PROCESSES (Top CPU consumers)

PID     Process                      CPU%    Memory      Disk      Network   CPU Hist   Mem Hist
12345   Xcode                        45.2%   2.3 GB      12 MB/s   0.5 MB/s  ████▇▆▅▄   ▅▅▅▅▅▅▅▅
23456   Chrome Helper (Renderer)     23.4%   892 MB      0 MB/s    2.1 MB/s  ▃▄▅▆▇███   ▆▆▆▇▇▇▇▇
34567   kernel_task                  18.9%   1.2 GB      34 MB/s   0 MB/s    ▂▃▄▅▄▃▂▁   ████████
45678   Spotify                      12.3%   445 MB      0 MB/s    0.3 MB/s  ▅▆▇▆▅▄▃▄   ▃▃▃▃▃▃▃▃
56789   Terminal                     8.7%    234 MB      1 MB/s    0 MB/s    ▁▂▃▄▃▂▁▁   ▂▂▂▂▂▂▂▂

RECENTLY EXITED PROCESSES (showing 4 of 4)
Process                                  Occurrences     Last Seen
swift build                                      3x        2m ago
clang++                                         5x        5m ago
node                                            2x        8m ago
python3                                         1x       12m ago
```

### Thermal Monitoring
```
THERMAL STATUS

Thermal Pressure: Fair ⚠️
  System is moderately throttling performance

Temperature Sensors:
  CPU P-Core 1:    78.3°C  ████████████████░░░░  Warning
  CPU P-Core 2:    76.1°C  ███████████████░░░░░  Warning
  CPU E-Core:      62.1°C  ████████████░░░░░░░░  Normal
  GPU:             71.2°C  ██████████████░░░░░░  Elevated
  Memory:          58.0°C  ███████████░░░░░░░░░  Normal
  SSD:             45.3°C  █████████░░░░░░░░░░░  Normal

Fan Speed: 4200 RPM (65% max)
```

### Battery Status
```
BATTERY STATUS

State:           Discharging 🔋
Charge:          67% ████████████████░░░░░░░░
Time Remaining:  4h 23m
Health:          92% (Normal)
Cycle Count:     234

Power Draw:      -18.4W
Voltage:         12.84V
Temperature:     31.2°C

Charging History: ▅▄▃▂▁▁▂▃▄▅▆▇█████▇▆▅▄▃▂▁
```

## 🛠️ Technical Details

### Architecture Support

- **Apple Silicon**: Full support for M1, M2, M3 series
  - Efficiency cores (E-cores) and Performance cores (P-cores) tracking
  - ANE (Apple Neural Engine) power monitoring
  - Unified memory architecture metrics

- **Intel Macs**: Complete compatibility
  - Traditional CPU frequency scaling
  - Turbo Boost monitoring
  - Discrete GPU tracking (if available)

### Data Sources

PowerMetrics TUI leverages macOS's native `powermetrics` utility, providing:
- Hardware-level accuracy
- Minimal performance overhead
- Real-time sampling capabilities
- Access to exclusive Apple Silicon metrics

## 🤝 Contributing

Contributions are welcome! Whether it's:
- 🐛 Bug reports
- 💡 Feature suggestions
- 📖 Documentation improvements
- 🔧 Code contributions

Please feel free to:
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [tcell](https://github.com/gdamore/tcell) - Excellent terminal UI library for Go
- Powered by macOS `powermetrics` - Apple's powerful system monitoring utility
- Inspired by tools like `htop`, `btop`, and `vtop`

## 🔗 Links

- [Report Issues](https://github.com/sderosiaux/powermetrics-tui/issues)
- [Discussions](https://github.com/sderosiaux/powermetrics-tui/discussions)
- [Wiki](https://github.com/sderosiaux/powermetrics-tui/wiki)

---

<div align="center">
Made with ❤️ for the macOS community
</div>