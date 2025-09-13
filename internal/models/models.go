package models

import (
	"sync"
	"time"
)

// MetricsState holds all system metrics
type MetricsState struct {
	Mu sync.RWMutex

	// CPU interrupts
	IPICount        int
	TimerCount      int
	TotalInterrupts int

	// Per-CPU interrupt breakdown
	PerCPUInterrupts map[string]float64 // CPU identifier -> interrupt rate
	PerCPUIPIs       map[string]float64 // CPU identifier -> IPI rate
	PerCPUTimers     map[string]float64 // CPU identifier -> Timer rate

	// Per-CPU interrupt history for sparklines
	PerCPUInterruptHistory map[string][]float64 // CPU identifier -> interrupt history

	// Power metrics
	CPUPower    float64
	GPUPower    float64
	ANEPower    float64
	DRAMPower   float64
	SystemPower float64

	// CPU frequency
	ECoreFreq  []int
	PCoreFreq  []int
	GPUFreq    int
	AllCpuFreq map[int]int // Temporary storage for all CPU frequencies

	// CPU frequency history (per core)
	ECoreFreqHistory map[int][]float64 // Core index -> frequency history
	PCoreFreqHistory map[int][]float64 // Core index -> frequency history

	// Network
	NetworkIn  float64
	NetworkOut float64

	// Disk I/O
	DiskRead  float64
	DiskWrite float64

	// Battery
	BatteryCharge float64
	BatteryState  string

	// Thermal
	ThermalPressure string
	Temperature     map[string]float64

	// System memory
	MemoryUsed      float64
	MemoryAvailable float64
	SwapUsed        float64

	// Process tracking
	Processes []ProcessInfo
	ProcessCPUHistory map[int][]float64 // PID -> CPU history
	ProcessMemHistory map[int][]float64 // PID -> Memory history

	// Historical data (circular buffers, 120 samples)
	History      *HistoricalData
	LastUpdate   time.Time
	UpdateErrors int
}

// ProcessInfo represents a single process
type ProcessInfo struct {
	PID           int
	Name          string
	CPUPercent    float64
	MemoryMB      float64
	DiskMB        float64
	NetworkMB     float64
	CPUHistory    []float64 // Last 10 samples for sparkline
	MemoryHistory []float64 // Last 10 samples for sparkline
}

// HistoricalData stores time series data
type HistoricalData struct {
	IPIHistory        []int
	TimerHistory      []int
	TotalHistory      []int
	CPUPowerHistory   []float64
	GPUPowerHistory   []float64
	SystemHistory     []float64
	NetworkInHistory  []float64
	NetworkOutHistory []float64
	DiskReadHistory   []float64
	DiskWriteHistory  []float64
	BatteryHistory    []float64
	TempHistory       []float64
	MemoryHistory     []float64
	MaxHistory        int
}

// NewMetricsState creates a new MetricsState with initialized history
func NewMetricsState() *MetricsState {
	return &MetricsState{
		Temperature: make(map[string]float64),
		ProcessCPUHistory: make(map[int][]float64),
		ProcessMemHistory: make(map[int][]float64),
		ECoreFreqHistory: make(map[int][]float64),
		PCoreFreqHistory: make(map[int][]float64),
		PerCPUInterrupts: make(map[string]float64),
		PerCPUIPIs: make(map[string]float64),
		PerCPUTimers: make(map[string]float64),
		PerCPUInterruptHistory: make(map[string][]float64),
		History: &HistoricalData{
			IPIHistory:        make([]int, 0, 120),
			TimerHistory:      make([]int, 0, 120),
			TotalHistory:      make([]int, 0, 120),
			CPUPowerHistory:   make([]float64, 0, 120),
			GPUPowerHistory:   make([]float64, 0, 120),
			SystemHistory:     make([]float64, 0, 120),
			NetworkInHistory:  make([]float64, 0, 120),
			NetworkOutHistory: make([]float64, 0, 120),
			DiskReadHistory:   make([]float64, 0, 120),
			DiskWriteHistory:  make([]float64, 0, 120),
			BatteryHistory:    make([]float64, 0, 120),
			TempHistory:       make([]float64, 0, 120),
			MemoryHistory:     make([]float64, 0, 120),
			MaxHistory:        120,
		},
	}
}

// AddToHistory adds a value to a historical data slice
func AddToHistory(history []float64, value float64, max int) []float64 {
	history = append(history, value)
	if len(history) > max {
		history = history[1:]
	}
	return history
}

// AddToIntHistory adds an int value to a historical data slice
func AddToIntHistory(history []int, value int, max int) []int {
	history = append(history, value)
	if len(history) > max {
		history = history[1:]
	}
	return history
}