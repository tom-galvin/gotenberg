package model

import (
	"tomgalvin.uk/phogoprint/internal/printer"
)

type DeviceInfoResponse struct {
	FirmwareVersion string
	State string
	BatteryLevel int
}

func FromDeviceInfo(i printer.DeviceInfo) DeviceInfoResponse {
	return DeviceInfoResponse{
		FirmwareVersion: i.FirmwareVersion,
		BatteryLevel: i.BatteryLevel,
		State: i.State.String(),
	}
}
