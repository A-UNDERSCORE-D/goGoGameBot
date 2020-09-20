package util

// TransportStatus is an int alias that indicates the status of a Transport
type TransportStatus int

// Various statuses for transports
const (
	Unknown TransportStatus = iota
	Running
	Stopped
)
