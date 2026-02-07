package types

// ProcessingMode represents the Cumulocity processing mode
type ProcessingMode string

// Cumulocity Processing Mode header
var HeaderProcessingMode = "X-Cumulocity-Processing-Mode"

// Available processing modes
const (
	ProcessingModePersistent ProcessingMode = "PERSISTENT"
	ProcessingModeTransient  ProcessingMode = "TRANSIENT"
	ProcessingModeQuiescent  ProcessingMode = "QUIESCENT"
	ProcessingModeCEP        ProcessingMode = "CEP"
)

// Legacy string variables for backward compatibility
var (
	ProcessingModePersistentStr = string(ProcessingModePersistent)
	ProcessingModeTransientStr  = string(ProcessingModeTransient)
	ProcessingModeQuiescentStr  = string(ProcessingModeQuiescent)
	ProcessingModeCEPStr        = string(ProcessingModeCEP)
)
