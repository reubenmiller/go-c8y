package microservice

import c8y "github.com/reubenmiller/go-c8y"

// AgentManagedObject is the agent representation of the microservice which is stored in Inventory
type AgentManagedObject struct {
	c8y.ManagedObject

	AgentConfiguration       *AgentConfiguration `json:"c8y_Configuration,omitempty"`
	AgentInformation         *AgentInformation   `json:"c8y_Hardware,omitempty"`
	AgentSupportedOperations []string            `json:"c8y_SupportedOperations,omitempty"`

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
	SerialNumber string `json:"serialNumber,omitempty"`
	Model        string `json:"model,omitempty"`
	Revision     string `json:"revision,omitempty"`
}

// AgentSupportedOperations is a list of operations which are supported by the agent
type AgentSupportedOperations []string
