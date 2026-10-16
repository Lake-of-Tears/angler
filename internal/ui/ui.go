package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
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
)

func PrintBanner() {
	banner := `
   🎣  ___   _  _   ___  _     ___  ___ 
      /   \ | \| | / __|| |   | __|| _ \
     | - - || .` + "`" + ` || (_ || |__ | _| |   /
     |_|_|_||_|\_| \___||____||___||_|_\
    Lake of Tears CLI — Cast. Query. Reel.
`
	fmt.Println(lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(banner))
}

func PrintSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", SuccessStyle.Render("✔"), msg)
}

func PrintInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", TitleStyle.Render("ℹ"), msg)
}

func PrintWarning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", WarningStyle.Render("▲"), msg)
}

func PrintError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "%s %s\n", ErrorStyle.Render("✖"), msg)
}
