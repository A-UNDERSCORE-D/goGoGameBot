package util

type TransportStatus int

// Various statuses for transports
const (
	Unknown TransportStatus = iota
	Running
	Stopped
)
