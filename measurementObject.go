package c8y

// MeasurementObject todo
type MeasurementObject struct {
	ID     string `json:"id"`
	Source struct {
		Self string `json:"self"`
		ID   string `json:"id"`
	} `json:"source"`
	Type string `json:"type"`
	Self string `json:"self"`
	// Time string `json:"time"`
	Time Timestamp `json:"time"`
}
