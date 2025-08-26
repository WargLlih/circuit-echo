package tui

import (
	"circuit-echo/portconfig"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/IonicHealthUsa/ionlog"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"go.bug.st/serial"
)

var (
	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#7C3AED")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Bold(true)

	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED"))

	normalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#666666"))
)

type model struct {
	serialViewport viewport.Model
	logViewport    viewport.Model
	textInput      textinput.Model
	serialContent  *strings.Builder
	logContent     *strings.Builder
	port           serial.Port
	width          int
	height         int
	focused        int
	serialData     <-chan string
	logData        <-chan string
	cancel         context.CancelFunc
}

func SetupConfiguration() (*portconfig.Config, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to list serial ports: %v", err)
	}

	if len(ports) == 0 {
		return nil, fmt.Errorf("no serial ports found")
	}

	// Create port options for huh
	portOptions := make([]huh.Option[string], len(ports))
	for i, port := range ports {
		portOptions[i] = huh.NewOption(port, port)
	}

	var portConfig portconfig.Config

	// Predefined baud rates
	baudRateOptions := []huh.Option[int]{
		huh.NewOption("9600", 9600),
		huh.NewOption("19200", 19200),
		huh.NewOption("38400", 38400),
		huh.NewOption("57600", 57600),
		huh.NewOption("115200", 115200),
		huh.NewOption("230400", 230400),
		huh.NewOption("460800", 460800),
		huh.NewOption("921600", 921600),
	}

	// Data bits options
	dataBitsOptions := []huh.Option[int]{
		huh.NewOption("5", 5),
		huh.NewOption("6", 6),
		huh.NewOption("7", 7),
		huh.NewOption("8", 8),
	}

	// Stop bits options
	stopBitsOptions := []huh.Option[int]{
		huh.NewOption("1", 1),
		huh.NewOption("2", 2),
	}

	// Parity options
	parityOptions := []huh.Option[string]{
		huh.NewOption("None", "N"),
		huh.NewOption("Odd", "O"),
		huh.NewOption("Even", "E"),
		huh.NewOption("Mark", "M"),
		huh.NewOption("Space", "S"),
	}

	// Custom baud rate input
	var customBaudRate string
	var useCustomBaudRate bool

	// Create the form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Serial Port").
				Options(portOptions...).
				Value(&portConfig.SelectedPort),

			huh.NewSelect[int]().
				Title("Select Baud Rate").
				Options(baudRateOptions...).
				Value(&portConfig.BaudRate),

			huh.NewConfirm().
				Title("Use custom baud rate?").
				Value(&useCustomBaudRate),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Enter custom baud rate").
				Value(&customBaudRate).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("baud rate cannot be empty")
					}
					if _, err := strconv.Atoi(str); err != nil {
						return fmt.Errorf("invalid baud rate: must be a number")
					}
					return nil
				}),
		).WithHideFunc(func() bool { return !useCustomBaudRate }),

		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Data Bits").
				Options(dataBitsOptions...).
				Value(&portConfig.DataBits).
				WithHeight(4),

			huh.NewSelect[int]().
				Title("Stop Bits").
				Options(stopBitsOptions...).
				Value(&portConfig.StopBits).
				WithHeight(3),

			huh.NewSelect[string]().
				Title("Parity").
				Options(parityOptions...).
				Value(&portConfig.Parity).
				WithHeight(6),
		),
	)

	// Run the form
	err = form.Run()
	if err != nil {
		return nil, fmt.Errorf("form error: %v", err)
	}

	// Handle custom baud rate
	if useCustomBaudRate && customBaudRate != "" {
		customRate, err := strconv.Atoi(customBaudRate)
		if err != nil {
			return nil, fmt.Errorf("invalid custom baud rate: %v", err)
		}
		portConfig.BaudRate = customRate
	}

	return &portConfig, nil
}

func InitialModel(port serial.Port, serialDataChannel <-chan string, logDataChannel <-chan string, cancel context.CancelFunc) model {
	ti := textinput.New()
	ti.Placeholder = "Type your message and press Enter to send..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	// Initialize viewports
	serialVp := viewport.New(50, 20)
	logVp := viewport.New(50, 20)

	m := model{
		textInput:      ti,
		serialViewport: serialVp,
		logViewport:    logVp,
		serialContent:  &strings.Builder{},
		logContent:     &strings.Builder{},
		port:           port,
		focused:        0,
		serialData:     serialDataChannel,
		logData:        logDataChannel,
		cancel:         cancel,
	}

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForSerialData(),
		m.waitForLogMessage(),
	)
}

type serialDataMsg struct {
	data string
}

type logMsg struct {
	message string
}

func (m model) waitForSerialData() tea.Cmd {
	return func() tea.Msg {
		data := <-m.serialData
		return serialDataMsg{data: data}
	}
}

func (m model) waitForLogMessage() tea.Cmd {
	return func() tea.Msg {
		data := <-m.logData
		return logMsg{message: data}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Resize viewports
		viewportHeight := ((m.height - 11) / 2)
		viewportWidth := (m.width - 2)

		m.logViewport.Height = viewportHeight - 5
		m.serialViewport.Height = viewportHeight + 5

		m.serialViewport.Width = viewportWidth
		m.logViewport.Width = viewportWidth
		m.textInput.Width = viewportWidth - 3

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.port == nil {
				return m, tea.Quit
			}

			if err := m.port.Close(); err != nil {
				ionlog.Errorf("Failed to close serial port: %v", err)
				return m, tea.Quit
			}

			if m.cancel != nil {
				m.cancel()
			}

			return m, tea.Quit

		case "tab":
			m.focused = (m.focused + 1) % 3
			if m.focused == 0 {
				m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}

		case "enter":
			if !(m.focused == 0 && m.textInput.Value() != "") {
				break
			}

			data := m.textInput.Value() + "\r\n"

			if m.port == nil {
				ionlog.Error("Serial port is not open, please restart the application")
				break
			}

			_, err := m.port.Write([]byte(data))
			if err != nil {
				ionlog.Errorf("Failed to write to serial port: %v", err)
				break
			}

			m.serialContent.WriteString(fmt.Sprintf("> %s", data))
			m.serialViewport.SetContent(m.serialContent.String())
			m.serialViewport.GotoBottom()
			m.textInput.SetValue("")
		}

	case serialDataMsg:
		m.serialContent.WriteString(msg.data)
		m.serialViewport.SetContent(m.serialContent.String())
		m.serialViewport.GotoBottom()

		ionlog.Infof("Serial Data: % 02x", []byte(msg.data))

		cmds = append(cmds, m.waitForSerialData())

	case logMsg:
		m.logContent.WriteString(msg.message)
		m.logViewport.SetContent(m.logContent.String())
		m.logViewport.GotoBottom()
		cmds = append(cmds, m.waitForLogMessage())
	}

	var cmd tea.Cmd
	switch m.focused {
	case 0:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		m.serialViewport, cmd = m.serialViewport.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		m.logViewport, cmd = m.logViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	// Serial I/O section
	serialTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		Bold(true).
		Render("Serial I/O")

	serialStyle := normalBorderStyle
	if m.focused == 1 {
		serialStyle = focusedBorderStyle
	}
	serialBox := serialStyle.Render(m.serialViewport.View())

	// Log section
	logTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true).
		Render("System Logs")

	logStyle := normalBorderStyle
	if m.focused == 2 {
		logStyle = focusedBorderStyle
	}

	logBox := logStyle.Render(m.logViewport.View())

	// Input section
	inputTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true).
		Render("Send Data:")

	inputStyle := normalBorderStyle
	if m.focused == 0 {
		inputStyle = focusedBorderStyle
	}
	inputBox := inputStyle.Render(m.textInput.View())

	// Help text
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("Tab: Switch focus • Enter: Send message • Ctrl+C/Esc: Quit")

	// Combine everything with proper spacing
	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		logTitle,
		logBox,
		serialTitle,
		serialBox,
		inputTitle,
		inputBox,
		help,
	)
}
