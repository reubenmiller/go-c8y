package c8y

// Operation todo
type Operation struct {
	ID           string    `json:"id"`
	CreationTime Timestamp `json:"creationTime"`
	DeviceID     string    `json:"deviceId"`
	Type         string    `json:"type"`
	DeviceName   string    `json:"deviceName"`
	Status       string    `json:"status"`
	Description  string    `json:"description"`
	Self         string    `json:"self"`
	EventID      string    `json:"eventId"`
}
