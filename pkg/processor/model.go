package processor

import "time"

// ServiceScan is the normalized representation of a scan observation.
type ServiceScan struct {
	IP         string
	Port       uint32
	Service    string
	ObservedAt time.Time
	Response   string
	MessageID  string
}
