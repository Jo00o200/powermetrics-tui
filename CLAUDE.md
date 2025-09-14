# PowerMetrics TUI - Technical Documentation

## Project Overview

PowerMetrics TUI is a real-time system monitoring dashboard for macOS that wraps the native `powermetrics` utility in a beautiful terminal UI. It uses a state machine-based parser to process the complex output from powermetrics and displays it in multiple specialized views.

## Architecture

### Core Components

```
┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│  powermetrics│────▶│   Parser    │────▶│ MetricsState │
│   (process)  │     │(State Machine)│     │   (Model)    │
└──────────────┘     └─────────────┘     └──────────────┘
                                                │
                                                ▼
                     ┌─────────────────────────────────┐
                     │         UI Layer                 │
                     │  ┌──────────────────────────┐   │
                     │  │ Views (10 specialized)   │   │
                     │  └──────────────────────────┘   │
                     └─────────────────────────────────┘
```

### Key Design Patterns

1. **State Machine Pattern**: Parser uses persistent state machine with dedicated handlers for each powermetrics section
2. **Observer Pattern**: UI observes MetricsState changes with mutex-protected concurrent access
3. **Strategy Pattern**: Different view types implement view-specific rendering strategies

## Parser State Machine

### States and Handlers

The parser operates as a persistent state machine that maintains context across line processing:

```go
// State transitions
StateWaitingForSample → StateInSample → StateProcessorUsage
                                    ├→ StateCPUInterrupts
                                    ├→ StateNetworkIO
                                    ├→ StateDiskIO
                                    ├→ StateThermalData
                                    ├→ StateGPUUsage
                                    ├→ StateBattery
                                    ├→ StateMemoryStats
                                    ├→ StatePowerMetrics
                                    ├→ StateFrequencies
                                    ├→ StateSFI
                                    └→ StateRunningTasks → StateTasksCoalition
                                                      └→ StateTasksSubprocess
```

### Section Routing

Section detection is centralized in `state_machine.go::routeSection()` using regex patterns:

```go
// Flexible regex patterns handle variations in powermetrics output
sectionProcessorUsage = regexp.MustCompile(`\*+\s*Processor usage\s*\*+`)
sectionGPUUsage = regexp.MustCompile(`\*+\s*GPU usage\s*\*+`)
// ... etc
```

### Critical Parser Behaviors

1. **DEAD_TASKS Handling**: Processes that die during sampling appear with empty names. Parser detects and tracks these in `RecentlyExited`.

2. **Line Consumption Issue**: When detecting section headers, the line must not be consumed without processing. Solution: centralized routing in state machine returns early after transition.

3. **Map Initialization**: All maps must be initialized in `ParseOutput()` to prevent nil pointer panics:
   - ProcessCPUHistory, ProcessMemHistory
   - CoalitionCPUHistory, CoalitionMemHistory
   - PerCPUInterruptHistory
   - ECoreFreqHistory, PCoreFreqHistory

## Data Models

### MetricsState Structure

```go
type MetricsState struct {
    Mu sync.RWMutex  // Protects concurrent access
    
    // Process tracking
    Processes []ProcessInfo
    Coalitions []ProcessCoalition
    RecentlyExited []ExitedProcessInfo  // Tracks dead processes
    
    // CPU metrics
    IPICount, TimerCount, TotalInterrupts int
    PerCPUInterrupts map[string]float64  // Per-CPU interrupt rates
    AllSeenCPUs map[string]bool          // Tracks all CPUs ever seen
    
    // Power metrics
    CPUPower, GPUPower, ANEPower, DRAMPower float64
    
    // Frequencies
    ECoreFreq, PCoreFreq []int
    GPUFreq int
    GPUFreqHistory []float64  // GPU frequency tracking
    
    // ... additional fields
}
```

### Process Classification

Processes are classified into three categories:
1. **Coalitions**: Parent process groups (identified by matching PID and coalition ID)
2. **Subprocesses**: Child processes within coalitions
3. **Dead Tasks**: Processes that terminated during sampling

## UI Components

### View System

10 specialized views, each optimized for specific metrics:
1. Interrupts - Per-CPU breakdown with sparklines
2. Power - Power consumption bars and history
3. Frequency - CPU/GPU frequencies with history
4. Processes - Top processes with sparklines
5. Network - I/O statistics
6. Disk - Read/write metrics
7. Thermal - Temperature and pressure
8. Battery - Charge and health
9. System - Overall metrics
10. Combined - All metrics in one view

### Visualization Techniques

1. **Sparklines**: Unicode block characters (▁▂▃▄▅▆▇█) for history visualization
2. **Bar Graphs**: Proportional bars with auto-scaling
3. **Color Coding**:
   - Red: Critical (>80% usage, >2000 interrupts/s)
   - Yellow: Warning (50-80% usage, 1000-2000 interrupts/s)
   - Green: Normal
   - Gray: Idle/inactive

### Special UI Features

1. **Fixed-Range Sparklines**: Battery uses 0-100% range via `DrawSparklineWithRange()`
2. **GPU Persistence**: GPU frequency always shows, grayed when idle
3. **Process History**: Individual CPU/memory sparklines per process

## Known Issues and Solutions

### Issue 1: DEAD_TASKS Problem
**Symptom**: Processes showing empty names in powermetrics output
**Cause**: Processes terminating during sampling period
**Solution**: Detect empty names, track in RecentlyExited with "Unknown Process (PID X)"

### Issue 2: Section Header Consumption
**Symptom**: Data not being parsed, state transitions consuming lines
**Cause**: Handlers returning new state after detecting section headers
**Solution**: Centralized section routing in state machine, early return after transition

### Issue 3: Battery Sparkline at 100%
**Symptom**: Sparkline shows flat line at bottom when battery at constant 100%
**Cause**: Auto-scaling sees no variation, shows baseline
**Solution**: Use fixed 0-100% range with `DrawSparklineWithRange()`

### Issue 4: GPU Frequency Disappearing
**Symptom**: GPU section disappears when idle
**Cause**: Conditional display only when frequency > 0
**Solution**: Always display, show "idle" state with gray color

## Testing Strategy

### Sample Files
- `sample_output_all.txt`: Main sample with all sections
- `sample_output_dead_processes.txt`: Sample with DEAD_TASKS

### Test Coverage
1. **Full parsing tests**: Use complete sample files
2. **Regex tests**: Use real lines from samples (not mocked)
3. **State machine tests**: Verify transitions and data flow
4. **History tests**: Verify sparkline data accumulation

### Running Tests
```bash
go test ./internal/parser -v
```

## Performance Considerations

1. **Sampling Rate**: Default 1000ms, can be reduced to 500ms for more responsive updates
2. **History Buffers**: Limited to 30-120 samples to prevent memory growth
3. **Mutex Usage**: RWMutex for MetricsState allows concurrent reads
4. **Process Filtering**: Top 20 processes by default to limit UI rendering

## Future Improvements

### Potential Enhancements
1. **Aggregated Sampling**: Sample at 200ms, update UI at 1s with averaged values
2. **Process Grouping**: Group Chrome/Electron helpers under parent
3. **Alert System**: Notify when thresholds exceeded
4. **Export Functionality**: Save metrics to CSV/JSON
5. **Configuration File**: User preferences for colors, thresholds, views

### Technical Debt
1. **Error Handling**: Some parsing errors silently ignored
2. **Test Coverage**: Need tests for UI components
3. **Documentation**: Inline code documentation could be improved

## Development Workflow

### Adding New Metrics
1. Add field to `MetricsState` in `models/models.go`
2. Create handler in `parser/handler_*.go`
3. Register handler in `state_machine.go`
4. Add regex pattern in `parser/parser.go`
5. Update UI view in `ui/views.go`
6. Add test using sample file

### Debugging Parser Issues
1. Enable debug output: `--debug` flag
2. Check `/tmp/powermetrics-debug.log`
3. Add temporary logging in handler `ProcessLine()` methods
4. Use `parser_test.go` to isolate parsing issues

### Common Pitfalls
1. **Forgetting map initialization**: Always init maps in `ParseOutput()`
2. **Consuming lines**: Ensure section detection doesn't consume data lines
3. **Regex patterns**: Test against actual powermetrics output variations
4. **Concurrent access**: Always use mutex when accessing MetricsState

## Code Style Guidelines

1. **No comments**: Code should be self-documenting
2. **Error handling**: Return errors up the chain, don't panic
3. **Testing**: Use real sample files, not mocked data
4. **UI updates**: Batch updates to reduce flicker
5. **Color usage**: Follow established color coding patterns

## Dependencies

- `github.com/gdamore/tcell/v2`: Terminal UI rendering
- `golang.org/x/sys/unix`: System calls for process management
- Standard library only for parsing and data structures

## Build and Release

### Building
```bash
go build -o powermetrics-tui
```

### Testing
```bash
go test ./...
```

### Linting
```bash
golangci-lint run
```

## Troubleshooting

### Common Issues

1. **"sudo required"**: powermetrics needs root access
2. **"command not found: powermetrics"**: Only available on macOS
3. **High CPU usage**: Reduce sampling interval or disable sparklines
4. **Garbled display**: Ensure terminal supports UTF-8
5. **Missing metrics**: Check `--samplers` flag includes required samplers

### Debug Commands

```bash
# Test powermetrics directly
sudo powermetrics --samplers all --show-all -i 1000 -n 1

# Check parser output
go test ./internal/parser -v -run TestParsePowerMetricsOutput

# Enable debug logging
./powermetrics-tui --debug
tail -f /tmp/powermetrics-debug.log
```

## Contact and Support

For issues or questions:
1. Check this documentation first
2. Review existing GitHub issues
3. Create detailed bug report with sample output
4. Include macOS version and hardware details

---

*Last updated: 2025*
*Version: 1.0.0*