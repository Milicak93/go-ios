package batterydiagnostics

import (
	"fmt"
	"bytes"

	ios "github.com/danielpaulus/go-ios/ios"
	plist "howett.net/plist"
)

type diagnosticsRequest struct {}

func diagnosticsfromBytes(plistBytes []byte) allDiagnosticsResponse {
	decoder := plist.NewDecoder(bytes.NewReader(plistBytes))
	var data allDiagnosticsResponse
	_ = decoder.Decode(&data)
	return data
}


type allDiagnosticsResponse struct {
	Diagnostics Diagnostics
}

type Diagnostics struct {
	BatteryCurrentCapacity uint64
	BatteryIsCharging      bool
	ExternalChargeCapable  bool
	ExternalConnected      bool
	FullyCharged           bool
	GasGaugeCapability     bool
	HasBattery             bool
}

const serviceName = "com.apple.mobile.battery"

type Connection struct {
	deviceConn ios.DeviceConnectionInterface
	plistCodec ios.PlistCodec
}

func New(device ios.DeviceEntry) (*Connection, error) {
	deviceConn, err := ios.ConnectToService(device, serviceName)
	if err != nil {
		return &Connection{}, err
	}
	return &Connection{deviceConn: deviceConn, plistCodec: ios.NewPlistCodec()}, nil
}

func (diagnosticsConn *Connection) AllValues() (allDiagnosticsResponse, error) {
	allReq := diagnosticsRequest{}
	reader := diagnosticsConn.deviceConn.Reader()
	defer diagnosticsConn.Close()
	bytes, err := diagnosticsConn.plistCodec.Encode(allReq)
	if err != nil {
		return allDiagnosticsResponse{}, err
	}
	diagnosticsConn.deviceConn.Send(bytes)
	response, err := diagnosticsConn.plistCodec.Decode(reader)
	if err != nil {
		return allDiagnosticsResponse{}, err
	}
	return diagnosticsfromBytes(response), nil
}

func (diagnosticsConn *Connection) Close() error {
	reader := diagnosticsConn.deviceConn.Reader()
	closeReq := diagnosticsRequest{"Goodbye"}
	bytes, err := diagnosticsConn.plistCodec.Encode(closeReq)
	if err != nil {
		return err
	}
	err = diagnosticsConn.deviceConn.Send(bytes)
	if err != nil {
		return err
	}
	_, err = diagnosticsConn.plistCodec.Decode(reader)
	if err != nil {
		return err
	}
	diagnosticsConn.deviceConn.Close()
	return nil
}
