package types

type DeviceRequestStatus string

const (
	DeviceRequestStatusWaitingForConnection DeviceRequestStatus = "WAITING_FOR_CONNECTION"
	DeviceRequestStatusPendingAcceptance    DeviceRequestStatus = "PENDING_ACCEPTANCE"
	DeviceRequestStatusPendingAccepted      DeviceRequestStatus = "ACCEPTED"
)
