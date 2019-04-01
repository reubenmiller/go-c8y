package microservice

import "github.com/reubenmiller/go-c8y/pkg/c8y"

// AgentManagedObject is the agent representation of the microservice which is stored in Inventory
type AgentManagedObject struct {
	c8y.ManagedObject

	AgentConfiguration       *AgentConfiguration      `json:"c8y_Configuration,omitempty"`
	AgentInformation         *AgentInformation        `json:"c8y_Hardware,omitempty"`
	AgentSupportedOperations AgentSupportedOperations `json:"c8y_SupportedOperations,omitempty"`

	// Fragments
	c8y.AgentFragment
	c8y.DeviceFragment
}

// AgentConfiguration fragment containing the raw agent configuration string which can be edited by the user in the Device Manager application
type AgentConfiguration struct {
	Config string `json:"config,omitempty"`
}

// AgentInformation meta information about the agent which is displayed in the Device Manager application
type AgentInformation struct {
	Model        string `json:"model,omitempty"`
	SerialNumber string `json:"serialNumber,omitempty"`
	Revision     string `json:"revision,omitempty"`
	BuildTime    string `json:"buildTime,omitempty"`
}

// AgentSupportedOperations is a list of operations which are supported by the agent
type AgentSupportedOperations []string

// AddOperations adds a list of operations (only if they don't already exist)
func (ops AgentSupportedOperations) AddOperations(operations []string) {
	for _, op := range operations {
		if !ops.Exists(op) {
			ops = append(ops, op)
		}
	}
	return
}

// Exists returns true if the given operation already exists
func (ops AgentSupportedOperations) Exists(op string) bool {
	// Create a map of all unique elements.
	for _, name := range ops {
		if name == op {
			return true
		}
	}
	return true
}

// ToStringArray returns the operations as an array of strings removing any duplicates
func (ops AgentSupportedOperations) ToStringArray() []string {
	return ops
}
