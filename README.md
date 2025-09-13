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
- **CPU Metrics**: Track interrupts (IPI, Timer), frequencies, and usage patterns
- **Power Analytics**: Monitor CPU, GPU, ANE (Neural Engine), and DRAM power consumption in real-time
- **Thermal Management**: View temperature sensors and thermal pressure states
- **Battery Intelligence**: Track charge levels, power draw, and battery health
- **Process Insights**: Identify resource-hungry applications at a glance
- **I/O Statistics**: Monitor network and disk activity
- **Memory Usage**: Track RAM and swap utilization

### 🎨 Beautiful Interface
- **10 Specialized Views**: Each metric category has its own optimized display
- **Real-time Visualizations**: Live-updating bar charts and sparkline graphs
- **120-Sample History**: Track trends over time with historical data
- **Smart Color Coding**: Instant visual feedback on system state
- **Responsive Design**: Adapts to your terminal size

### 🤝 User-Friendly
- **Help System**: Built-in descriptions for technical terms (toggle with 'h')
- **Quick Navigation**: Jump to any view with number keys (1-9, 0)
- **Cross-Architecture**: Works seamlessly on Apple Silicon (M1/M2/M3) and Intel Macs
- **Auto-Detection**: Intelligently adapts to your Mac's capabilities

## 📥 Installation

### Prerequisites
- macOS (any version with `powermetrics` utility)
- Go 1.21 or later
- Terminal with UTF-8 support

### Quick Install

```bash
# Clone the repository
git clone https://github.com/yourusername/powermetrics-tui.git
cd powermetrics-tui

# Build the application
go build -o powermetrics-tui

# Make it globally accessible (optional)
sudo cp powermetrics-tui /usr/local/bin/
```

### Install from Source

```bash
go install github.com/yourusername/powermetrics-tui@latest
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

1. **Interrupts** - CPU interrupt statistics
2. **Power** - Power consumption metrics
3. **Frequency** - CPU/GPU clock speeds
4. **Processes** - Top processes by resource usage
5. **Network** - Network I/O statistics
6. **Disk** - Disk I/O statistics
7. **Thermal** - Temperature monitoring
8. **Battery** - Battery and charging status
9. **System** - Overall system metrics
0. **Combined** - All metrics in one view

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

## 📸 Screenshots

### Power Consumption View
```
POWER CONSUMPTION (Energy usage - affects battery life)

CPU:     3421.2 mW  ████████████████░░░░░░░░░░░░
  Processor power consumption

GPU:      892.3 mW  ████░░░░░░░░░░░░░░░░░░░░░░░░
  Graphics processor power

ANE:       12.5 mW  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░
  Apple Neural Engine - AI/ML accelerator
```

### Thermal Monitoring
```
THERMAL STATUS

Thermal Pressure: Nominal ✅

Temperature Sensors:
  CPU P-Core 1:    42.3°C  ████████░░░░░░░░
  CPU E-Core:      38.1°C  ███████░░░░░░░░░
  GPU:             45.2°C  █████████░░░░░░░
  Memory:          41.0°C  ████████░░░░░░░░
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

- [Report Issues](https://github.com/yourusername/powermetrics-tui/issues)
- [Discussions](https://github.com/yourusername/powermetrics-tui/discussions)
- [Wiki](https://github.com/yourusername/powermetrics-tui/wiki)

---

<div align="center">
Made with ❤️ for the macOS community
</div>