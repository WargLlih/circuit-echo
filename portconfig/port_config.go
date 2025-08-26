package portconfig

import (
	"context"
	"time"

	"github.com/IonicHealthUsa/ionlog"
	"go.bug.st/serial"
)

type Config struct {
	SelectedPort string
	BaudRate     int
	DataBits     int
	StopBits     int
	Parity       string
}

func OpenSerialPort(config Config) (serial.Port, error) {
	// Configure serial port
	var parity serial.Parity
	switch config.Parity {
	case "N":
		parity = serial.NoParity
	case "O":
		parity = serial.OddParity
	case "E":
		parity = serial.EvenParity
	case "M":
		parity = serial.MarkParity
	case "S":
		parity = serial.SpaceParity
	}

	var stopBits serial.StopBits
	switch config.StopBits {
	case 1:
		stopBits = serial.OneStopBit
	case 2:
		stopBits = serial.TwoStopBits
	}

	mode := &serial.Mode{
		BaudRate: config.BaudRate,
		DataBits: config.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	}

	return serial.Open(config.SelectedPort, mode)
}

func SerialReader(ctx context.Context, port serial.Port, serialDataChannel chan<- string) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			ionlog.Info("Serial reader stopping")
			return

		default:
			n, err := port.Read(buf)
			if err != nil {
				ionlog.Errorf("Error reading from serial port: %v", err)
				ionlog.Errorf("Please restart the application to recover.")
				return
			}

			if n > 0 {
				select {
				case serialDataChannel <- string(buf[:n]):
				case <-time.After(1 * time.Second):
					ionlog.Warn("Serial data channel is full, dropping data")
				}
			}
		}
	}
}
