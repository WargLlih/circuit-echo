package main

import (
	"circuit-echo/logwriter"
	"circuit-echo/portconfig"
	"circuit-echo/tui"
	"context"
	"fmt"
	"os"

	"github.com/IonicHealthUsa/ionlog"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	ionlog.SetAttributes(
		ionlog.WithWriters(
			ionlog.CustomOutput(logwriter.Logger()),
		),
	)

	ionlog.Start()
	defer ionlog.Stop()

	portConfig, err := tui.SetupConfiguration()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		return
	}

	if portConfig == nil {
		fmt.Fprintf(os.Stderr, "No configuration selected, exiting.\n")
		return
	}

	port, err := portconfig.OpenSerialPort(*portConfig)
	defer func() {
		if err := port.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close serial port: %v\n", err)
			return
		}
	}()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open serial port: %v\n", err)
		return
	}

	serialData := make(chan string, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	go portconfig.SerialReader(ctx, port, serialData)

	p := tea.NewProgram(
		tui.InitialModel(
			port,
			serialData,
			logwriter.Messages(),
			cancel,
		),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		return
	}
}

// // Color codes
// const (
// 	Reset  = "\033[0m"
// 	Red    = "\033[31m"
// 	Green  = "\033[32m"
// 	Yellow = "\033[33m"
// 	Blue   = "\033[34m"
// 	Cyan   = "\033[36m"
// 	Gray   = "\033[90m"
// )
//
// func colorLog(line string) string {
// 	// Split into 3 parts: time, level, message
// 	parts := strings.SplitN(line, " ", 3)
// 	if len(parts) < 3 {
// 		return line // fallback if malformed
// 	}
//
// 	timePart := parts[0]
// 	levelPart := parts[1]
// 	msgPart := parts[2]
//
// 	// Colorize
// 	coloredTime := Gray + timePart + Reset
// 	coloredLevel := ""
// 	switch levelPart {
// 	case "INFO":
// 		coloredLevel = Green + levelPart + Reset
// 	case "WARN":
// 		coloredLevel = Yellow + levelPart + Reset
// 	case "ERRO":
// 		coloredLevel = Red + levelPart + Reset
// 	default:
// 		coloredLevel = Cyan + levelPart + Reset
// 	}
// 	coloredMsg := Blue + msgPart + Reset
//
// 	return fmt.Sprintf("%s %s %s", coloredTime, coloredLevel, coloredMsg)
// }
//
