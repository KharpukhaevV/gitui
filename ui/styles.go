package ui

import (
	"github.com/KharpukhaevV/gitui/utils"
	"github.com/charmbracelet/lipgloss"
)

// Стили для приложения
var (
	AppStyle = lipgloss.NewStyle().Padding(utils.DefaultPadding, 2)

	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	AccountItemStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Margin(0, 1)

	ActiveAccountStyle = AccountItemStyle.Copy().
				Foreground(lipgloss.Color("#25A065")).
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)

	AddAccountStyle = AccountItemStyle.Copy().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	ActiveAddAccountStyle = AddAccountStyle.Copy().
				Foreground(lipgloss.Color("#25A065")).
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)

	InputStyle = lipgloss.NewStyle().Width(utils.DefaultInputWidth).BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, utils.DefaultPadding)

	FormTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#25A065")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#25A065"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)
