package parser

import (
	"fmt"
	"regexp"

	"powermetrics-tui/internal/models"
)

// ParserState represents the current state of the powermetrics parser
type ParserState int

const (
	StateWaitingForSample ParserState = iota
	StateInSample
	StateProcessorUsage
	StateCPUInterrupts
	StatePowerMetrics
	StateFrequencies
	StateNetworkIO
	StateDiskIO
	StateMemoryStats
	StateThermalData
	StateGPUUsage
	StateBattery
	StateSFI
	StateRunningTasks
	StateTasksCoalition
	StateTasksSubprocess
	StateError
)

// String returns a human-readable representation of the parser state
func (s ParserState) String() string {
	switch s {
	case StateWaitingForSample:
		return "WaitingForSample"
	case StateInSample:
		return "InSample"
	case StateProcessorUsage:
		return "ProcessorUsage"
	case StateCPUInterrupts:
		return "CPUInterrupts"
	case StatePowerMetrics:
		return "PowerMetrics"
	case StateFrequencies:
		return "Frequencies"
	case StateNetworkIO:
		return "NetworkIO"
	case StateDiskIO:
		return "DiskIO"
	case StateMemoryStats:
		return "MemoryStats"
	case StateThermalData:
		return "ThermalData"
	case StateGPUUsage:
		return "GPUUsage"
	case StateBattery:
		return "Battery"
	case StateSFI:
		return "SFI"
	case StateRunningTasks:
		return "RunningTasks"
	case StateTasksCoalition:
		return "TasksCoalition"
	case StateTasksSubprocess:
		return "TasksSubprocess"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// ParserContext holds the context data for the state machine
type ParserContext struct {
	// Current state
	State ParserState

	// Context data for parsing
	CurrentCPU     string  // Current CPU being parsed (for interrupts)
	CurrentCluster string  // Current cluster (E/P) for frequency parsing

	// Accumulator variables for current sample
	IPITotal       float64
	TimerTotal     float64
	InterruptsTotal float64

	// Process/coalition parsing context
	CurrentCoalition     *models.ProcessCoalition
	NewProcesses         []models.ProcessInfo
	NewCoalitions        []models.ProcessCoalition
	OrphanedSubprocesses []models.ProcessInfo

	// Debug/logging
	DebugEnabled bool
	SampleCount  int

	// Reference to the metrics state being updated
	MetricsState *models.MetricsState
}

// NewParserContext creates a new parser context
func NewParserContext(metricsState *models.MetricsState) *ParserContext {
	return &ParserContext{
		State:                StateWaitingForSample,
		MetricsState:         metricsState,
		NewProcesses:         make([]models.ProcessInfo, 0),
		NewCoalitions:        make([]models.ProcessCoalition, 0),
		OrphanedSubprocesses: make([]models.ProcessInfo, 0),
	}
}

// StateHandler interface for handling different parser states
type StateHandler interface {
	// ProcessLine processes a line in this state and returns the next state
	ProcessLine(ctx *ParserContext, line string) ParserState

	// Enter is called when entering this state
	Enter(ctx *ParserContext)

	// Exit is called when leaving this state
	Exit(ctx *ParserContext)

	// Name returns the name of this state handler
	Name() string
}

// StateMachine manages the parsing state transitions
type StateMachine struct {
	handlers map[ParserState]StateHandler
	context  *ParserContext
}

// NewStateMachine creates a new state machine with all handlers
func NewStateMachine(metricsState *models.MetricsState) *StateMachine {
	ctx := NewParserContext(metricsState)

	sm := &StateMachine{
		handlers: make(map[ParserState]StateHandler),
		context:  ctx,
	}

	// Register all state handlers
	sm.RegisterHandler(StateWaitingForSample, &WaitingForSampleHandler{})
	sm.RegisterHandler(StateInSample, &InSampleHandler{})
	sm.RegisterHandler(StateProcessorUsage, &ProcessorUsageHandler{})
	sm.RegisterHandler(StateCPUInterrupts, &CPUInterruptsHandler{})
	sm.RegisterHandler(StatePowerMetrics, &PowerMetricsHandler{})
	sm.RegisterHandler(StateFrequencies, &FrequenciesHandler{})
	sm.RegisterHandler(StateNetworkIO, &NetworkIOHandler{})
	sm.RegisterHandler(StateDiskIO, &DiskIOHandler{})
	sm.RegisterHandler(StateMemoryStats, &MemoryStatsHandler{})
	sm.RegisterHandler(StateThermalData, &ThermalDataHandler{})
	sm.RegisterHandler(StateGPUUsage, &GPUUsageHandler{})
	sm.RegisterHandler(StateBattery, &BatteryHandler{})
	sm.RegisterHandler(StateSFI, &SFIHandler{})
	sm.RegisterHandler(StateRunningTasks, &RunningTasksHandler{})
	sm.RegisterHandler(StateError, &ErrorHandler{})

	return sm
}

// RegisterHandler registers a state handler
func (sm *StateMachine) RegisterHandler(state ParserState, handler StateHandler) {
	sm.handlers[state] = handler
}

// ProcessLine processes a single line and manages state transitions
func (sm *StateMachine) ProcessLine(line string) error {
	// Global check for new sample - should reset to waiting state from any state
	if sm.context.State != StateWaitingForSample && IsNewSample(line) {
		sm.TransitionTo(StateWaitingForSample)
		// Let the WaitingForSample handler process this line
	}

	// Check for section headers and route appropriately (except in WaitingForSample state)
	if sm.context.State != StateWaitingForSample && IsSection(line) {
		// Route to appropriate section handler
		nextState := sm.routeSection(line)
		if nextState != sm.context.State {
			sm.TransitionTo(nextState)
			return nil // Don't process this line further
		}
	}

	handler, exists := sm.handlers[sm.context.State]
	if !exists {
		return fmt.Errorf("no handler for state %s", sm.context.State)
	}

	// Process the line and get the next state
	nextState := handler.ProcessLine(sm.context, line)

	// Handle state transition if needed
	if nextState != sm.context.State {
		// Debug logging removed - no console output
		sm.TransitionTo(nextState)
	}

	return nil
}

// Section header patterns - flexible regex to handle variations
var (
	sectionProcessorUsage = regexp.MustCompile(`\*+\s*Processor usage\s*\*+`)
	sectionRunningTasks   = regexp.MustCompile(`\*+\s*Running tasks\s*\*+`)
	sectionInterrupts     = regexp.MustCompile(`\*+\s*Interrupt distribution\s*\*+`)
	sectionNetworkIO      = regexp.MustCompile(`\*+\s*Network activity\s*\*+`)
	sectionDiskIO         = regexp.MustCompile(`\*+\s*Disk activity\s*\*+`)
	sectionThermal        = regexp.MustCompile(`\*+\s*Thermal pressure\s*\*+`)
	sectionMemory         = regexp.MustCompile(`\*+\s*System memory\s*\*+`)
	sectionPowerMetrics   = regexp.MustCompile(`\*+\s*Power\s*\*+`)
	sectionFrequencies    = regexp.MustCompile(`\*+\s*Frequency\s*\*+`)
	sectionGPUUsage       = regexp.MustCompile(`\*+\s*GPU usage\s*\*+`)
	sectionBattery        = regexp.MustCompile(`\*+\s*Battery and backlight usage\s*\*+`)
	sectionSFI            = regexp.MustCompile(`\*+\s*Selective Forced Idle\s*\*+`)
)

// routeSection determines which state to transition to based on section header
func (sm *StateMachine) routeSection(line string) ParserState {
	// Use regex patterns for flexible section matching
	switch {
	case sectionProcessorUsage.MatchString(line):
		return StateProcessorUsage
	case sectionRunningTasks.MatchString(line):
		return StateRunningTasks
	case sectionInterrupts.MatchString(line):
		return StateCPUInterrupts
	case sectionNetworkIO.MatchString(line):
		return StateNetworkIO
	case sectionDiskIO.MatchString(line):
		return StateDiskIO
	case sectionThermal.MatchString(line):
		return StateThermalData
	case sectionMemory.MatchString(line):
		return StateMemoryStats
	case sectionPowerMetrics.MatchString(line):
		return StatePowerMetrics
	case sectionFrequencies.MatchString(line):
		return StateFrequencies
	case sectionGPUUsage.MatchString(line):
		return StateGPUUsage
	case sectionBattery.MatchString(line):
		return StateBattery
	case sectionSFI.MatchString(line):
		return StateSFI
	default:
		// For sections we don't have dedicated handlers yet, stay in InSample
		return StateInSample
	}
}

// TransitionTo transitions to a new state
func (sm *StateMachine) TransitionTo(newState ParserState) error {
	currentHandler, exists := sm.handlers[sm.context.State]
	if exists {
		currentHandler.Exit(sm.context)
	}

	sm.context.State = newState

	newHandler, exists := sm.handlers[newState]
	if !exists {
		return fmt.Errorf("no handler for state %s", newState)
	}

	newHandler.Enter(sm.context)
	return nil
}

// GetContext returns the current parser context
func (sm *StateMachine) GetContext() *ParserContext {
	return sm.context
}

// Reset resets the state machine to initial state
func (sm *StateMachine) Reset() {
	sm.TransitionTo(StateWaitingForSample)
	sm.context.IPITotal = 0
	sm.context.TimerTotal = 0
	sm.context.InterruptsTotal = 0
	sm.context.CurrentCPU = ""
	sm.context.CurrentCluster = ""
	sm.context.CurrentCoalition = nil
	sm.context.NewProcesses = make([]models.ProcessInfo, 0)
	sm.context.NewCoalitions = make([]models.ProcessCoalition, 0)
	sm.context.OrphanedSubprocesses = make([]models.ProcessInfo, 0)
}

// EnableDebug enables debug logging for state transitions
func (sm *StateMachine) EnableDebug(enabled bool) {
	sm.context.DebugEnabled = enabled
}

// FinalizeCurrentState forces the Exit method of the current state handler
// This is useful when parsing is complete and we need to commit data
func (sm *StateMachine) FinalizeCurrentState() {
	// If we're in RunningTasks state, call its Exit method to commit data
	if sm.context.State == StateRunningTasks {
		if handler, exists := sm.handlers[StateRunningTasks]; exists {
			handler.Exit(sm.context)
		}
	}
	// Reset to waiting state for next sample
	sm.TransitionTo(StateWaitingForSample)
}