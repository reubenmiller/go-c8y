package model

// SystemOption representation
type SystemOption struct {
	Category string `json:"category,omitempty"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
}

// SystemOptionCollection collection of system options
type SystemOptionCollection struct {
	Options []SystemOption `json:"options"`
}
