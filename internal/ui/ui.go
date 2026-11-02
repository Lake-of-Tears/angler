package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatYAML  = "yaml"
	FormatTOON  = "toon"
	FormatCSV   = "csv"
)

const (
	ExitOK              = 0
	ExitGeneralError    = 1
	ExitUsageError      = 2
	ExitAuthError       = 3
	ExitNotFoundError   = 4
	ExitConnectionError = 5
)

var (
	// Brand colors: Lake Blue & Emerald
	PrimaryColor   = lipgloss.Color("#00B4D8")
	SecondaryColor = lipgloss.Color("#0077B6")
	SuccessColor   = lipgloss.Color("#2EC4B6")
	WarningColor   = lipgloss.Color("#FF9F1C")
	DangerColor    = lipgloss.Color("#E71D36")
	MutedColor     = lipgloss.Color("#8D99AE")
	BorderColor    = lipgloss.Color("#023E8A")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SuccessColor)

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(WarningColor)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(DangerColor)

	MutedStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	// Configurable writers and runtime state for CLI and test harnesses
	OutWriter io.Writer = os.Stdout
	ErrWriter io.Writer = os.Stderr

	currentFormat  = FormatTable
	nonInteractive = false

	// Optional overrides for deterministic unit tests
	stdoutTTYOverride *bool
	stdinTTYOverride  *bool
)

// IsStdoutTTY reports whether standard output is connected to an interactive terminal.
func IsStdoutTTY() bool {
	if stdoutTTYOverride != nil {
		return *stdoutTTYOverride
	}
	f, ok := OutWriter.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// IsStdinTTY reports whether standard input is connected to an interactive terminal.
func IsStdinTTY() bool {
	if stdinTTYOverride != nil {
		return *stdinTTYOverride
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// SetTTYOverrides allows tests to simulate interactive or non-interactive terminals.
func SetTTYOverrides(stdoutTTY, stdinTTY *bool) {
	stdoutTTYOverride = stdoutTTY
	stdinTTYOverride = stdinTTY
}

// SetOutputFormat sets the active global output format.
func SetOutputFormat(format string) {
	if format == "" {
		currentFormat = FormatTable
		return
	}
	currentFormat = format
}

// GetOutputFormat returns the active global output format.
func GetOutputFormat() string {
	return currentFormat
}

// SetNonInteractive sets whether interactive terminal prompts are disabled.
func SetNonInteractive(val bool) {
	nonInteractive = val
}

// IsNonInteractive reports whether prompts should be bypassed.
func IsNonInteractive() bool {
	return nonInteractive || !IsStdinTTY() || !IsStdoutTTY()
}

// IsMachineOutput reports whether the output is structured machine output or non-TTY.
func IsMachineOutput() bool {
	switch currentFormat {
	case FormatJSON, FormatYAML, FormatTOON, FormatCSV:
		return true
	}
	return !IsStdoutTTY()
}

// StyleText renders text with the given lipgloss style only when interactive human output is active.
func StyleText(style lipgloss.Style, text string) string {
	if IsMachineOutput() {
		return text
	}
	return style.Render(text)
}

func infoDestination() io.Writer {
	if IsMachineOutput() {
		return ErrWriter
	}
	return OutWriter
}

func PrintBanner() {
	if IsMachineOutput() {
		return
	}
	banner := `
   🎣  ___   _  _   ___  _     ___  ___ 
      /   \ | \| | / __|| |   | __|| _ \
     | - - || .` + "`" + ` || (_ || |__ | _| |   /
     |_|_|_||_|\_| \___||____||___||_|_\
    Lake of Tears CLI — Cast. Query. Reel.
`
	fmt.Fprintln(OutWriter, lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(banner))
}

func PrintSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	icon := StyleText(SuccessStyle, "✔")
	fmt.Fprintf(infoDestination(), "%s %s\n", icon, msg)
}

func PrintInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	icon := StyleText(TitleStyle, "ℹ")
	fmt.Fprintf(infoDestination(), "%s %s\n", icon, msg)
}

func PrintWarning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	icon := StyleText(WarningStyle, "▲")
	fmt.Fprintf(infoDestination(), "%s %s\n", icon, msg)
}

func PrintError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	icon := StyleText(ErrorStyle, "✖")
	fmt.Fprintf(ErrWriter, "%s %s\n", icon, msg)
}
