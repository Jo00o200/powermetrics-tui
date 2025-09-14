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
	AllSeenCPUs      map[string]bool    // Track all CPUs ever seen for consistent display

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
	MaxECores  int         // Maximum number of E-cores ever seen
	MaxPCores  int         // Maximum number of P-cores ever seen

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
	Coalitions []ProcessCoalition
	ProcessCPUHistory map[int][]float64 // PID -> CPU history
	ProcessMemHistory map[int][]float64 // PID -> Memory history
	CoalitionCPUHistory map[int][]float64 // Coalition ID -> CPU history
	CoalitionMemHistory map[int][]float64 // Coalition ID -> Memory history

	// Recently exited processes tracking
	RecentlyExited []ExitedProcessInfo
	LastSeenPIDs   map[int]time.Time  // Track when each PID was last seen
	ProcessNames   map[int]string     // Track process names by PID (NOT coalition IDs)
	CoalitionNames map[int]string     // Track coalition names by Coalition ID (separate from PIDs)

	// Historical data (circular buffers, 120 samples)
	History      *HistoricalData
	LastUpdate   time.Time
	UpdateErrors int
}

// ProcessCoalition represents a process coalition (parent process group)
type ProcessCoalition struct {
	CoalitionID   int
	Name          string
	CPUPercent    float64
	MemoryMB      float64
	DiskMB        float64
	NetworkMB     float64
	Subprocesses  []ProcessInfo
	CPUHistory    []float64 // Last 10 samples for sparkline
	MemoryHistory []float64 // Last 10 samples for sparkline
}

// ProcessInfo represents a single process (subprocess within a coalition)
type ProcessInfo struct {
	PID           int
	Name          string
	CoalitionName string    // Name of parent coalition
	CPUPercent    float64
	MemoryMB      float64
	DiskMB        float64
	NetworkMB     float64
	CPUHistory    []float64 // Last 10 samples for sparkline
	MemoryHistory []float64 // Last 10 samples for sparkline
}

// ExitedProcessInfo represents a process that recently exited
type ExitedProcessInfo struct {
	Name         string
	PIDs         []int     // List of all PIDs that exited for this process name
	Occurrences  int       // Number of times this process has appeared and exited
	LastExitTime time.Time // When the process last exited
	FirstSeenTime time.Time // When we first saw this process name
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
		Coalitions: make([]ProcessCoalition, 0),
		CoalitionCPUHistory: make(map[int][]float64),
		CoalitionMemHistory: make(map[int][]float64),
		ECoreFreqHistory: make(map[int][]float64),
		PCoreFreqHistory: make(map[int][]float64),
		PerCPUInterrupts: make(map[string]float64),
		PerCPUIPIs: make(map[string]float64),
		PerCPUTimers: make(map[string]float64),
		AllSeenCPUs: make(map[string]bool),
		PerCPUInterruptHistory: make(map[string][]float64),
		RecentlyExited: make([]ExitedProcessInfo, 0),
		LastSeenPIDs: make(map[int]time.Time),
		ProcessNames: make(map[int]string),
		CoalitionNames: make(map[int]string),
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