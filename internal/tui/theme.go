package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	Accent  = lipgloss.Color("#7C3AED")
	Title   = lipgloss.NewStyle().Bold(true).Foreground(Accent).Render
	Info    = lipgloss.NewStyle().Foreground(lipgloss.Color("#93C5FD")).Render
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Render
	Warn    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Render
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Bold(true).Render
	Dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render
	Header  = lipgloss.NewStyle().
		Bold(true).
		Foreground(Accent).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 2).
		Render
)

func Confirm(title string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Evet").
		Negative("Hayır").
		Value(&ok).
		Run()
	return ok, err
}

func ConfirmOrAbort(title string) bool {
	ok, err := Confirm(title)
	if err != nil {
		return false
	}
	return ok
}

func SelectOne(title string, options []string, selected *string) error {
	if len(options) == 0 {
		return fmt.Errorf("seçilecek öğe yok")
	}
	return huh.NewSelect[string]().
		Title(title).
		Options(mapOptions(options)...).
		Value(selected).
		Run()
}

func SelectMany(title string, options []string, selected *[]string) error {
	if len(options) == 0 {
		return fmt.Errorf("seçilecek öğe yok")
	}
	return huh.NewMultiSelect[string]().
		Title(title).
		Description("Boşluk: seç / Enter: onayla").
		Options(mapOptions(options)...).
		Value(selected).
		Run()
}

func mapOptions(options []string) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(options))
	for _, o := range options {
		opts = append(opts, huh.NewOption(o, o))
	}
	return opts
}

func PromptText(title string, value *string) error {
	return huh.NewInput().Title(title).Value(value).Run()
}

func PromptInt(title string, value *int) error {
	var s string
	if err := huh.NewInput().Title(title).Value(&s).Run(); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return err
	}
	*value = v
	return nil
}

func PrintTitle(s string) {
	fmt.Println(Header(" " + s))
}

func PrintInfo(format string, args ...any) {
	fmt.Println(Info(fmt.Sprintf(format, args...)))
}

func PrintSuccess(format string, args ...any) {
	fmt.Println(Success(fmt.Sprintf(format, args...)))
}

func PrintWarn(format string, args ...any) {
	fmt.Println(Warn(fmt.Sprintf(format, args...)))
}

func PrintError(format string, args ...any) {
	fmt.Println(Error(fmt.Sprintf(format, args...)))
}

func PrintDim(format string, args ...any) {
	fmt.Println(Dim(fmt.Sprintf(format, args...)))
}
