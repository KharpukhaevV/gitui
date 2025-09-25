package ui

import (
	"fmt"
	"strings"

	"github.com/KharpukhaevV/gitui/models"
	"github.com/KharpukhaevV/gitui/utils"
	"github.com/charmbracelet/lipgloss"
)

// RenderAccountsScreen рендерит экран выбора аккаунтов
func RenderAccountsScreen(m *AppModel) string {
	doc := strings.Builder{}

	// Заголовок - центрируем по всей ширине
	title := TitleStyle.Render("GitHub Account Manager")
	centeredTitle := lipgloss.Place(m.Width, 1, lipgloss.Center, lipgloss.Center, title)
	doc.WriteString(centeredTitle + "\n\n")

	// Список аккаунтов
	var accountItems []string
	for i, accountName := range m.AccountsList {
		if i == m.SelectedAccount {
			if i == len(m.AccountsList)-1 {
				accountItems = append(accountItems, ActiveAddAccountStyle.Render(accountName))
			} else {
				accountItems = append(accountItems, ActiveAccountStyle.Render(accountName))
			}
		} else {
			if i == len(m.AccountsList)-1 {
				accountItems = append(accountItems, AddAccountStyle.Render(accountName))
			} else {
				accountItems = append(accountItems, AccountItemStyle.Render(accountName))
			}
		}
	}

	// Объединяем аккаунты вертикально
	accountsView := lipgloss.JoinVertical(lipgloss.Center, accountItems...)

	// Центрируем весь блок по горизонтали и вертикали
	centeredAccounts := lipgloss.Place(
		m.Width,
		m.Height-10, // Оставляем место для заголовка и инструкций
		lipgloss.Center,
		lipgloss.Center,
		accountsView,
	)
	doc.WriteString(centeredAccounts + "\n\n")

	// Сообщение
	if m.Message != "" {
		var style lipgloss.Style
		if m.MessageType == "success" {
			style = SuccessStyle
		} else {
			style = ErrorStyle
		}
		centeredMessage := lipgloss.Place(m.Width, 1, lipgloss.Center, lipgloss.Center, style.Render(m.Message))
		doc.WriteString(centeredMessage + "\n\n")
	}

	// Инструкции
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Render("Use ↑/↓ to navigate, Enter to select, q to quit")

	centeredInstructions := lipgloss.Place(m.Width, 1, lipgloss.Center, lipgloss.Center, instructions)
	doc.WriteString(centeredInstructions)

	return AppStyle.Render(doc.String())
}

// RenderReposScreen рендерит экран репозиториев
func RenderReposScreen(m *AppModel) string {
	doc := strings.Builder{}

	doc.WriteString(fmt.Sprintf("Account: %s\n", m.SelectedAccountPtr.Name))

	// Показываем путь develop directory
	devPath, exists, err := utils.CheckDevelopDir()
	if err == nil {
		status := "✅"
		if !exists {
			status = "❌"
		}
		doc.WriteString(fmt.Sprintf("Develop directory: %s %s\n\n", devPath, status))
	}

	if m.Loading {
		doc.WriteString(fmt.Sprintf("%s Loading repositories...\n\n", m.Spinner.View()))
	} else {
		doc.WriteString(m.List.View() + "\n\n")
	}

	// Сообщение
	if m.Message != "" {
		var style lipgloss.Style
		if m.MessageType == "success" {
			style = SuccessStyle
		} else {
			style = ErrorStyle
		}
		doc.WriteString(style.Render(m.Message) + "\n\n")
	}

	doc.WriteString("Press c to clone, r to refresh, esc to back, q to quit")

	return AppStyle.Render(doc.String())
}

// RenderAddAccountScreen рендерит экран добавления аккаунта
func RenderAddAccountScreen(m *AppModel) string {
	doc := strings.Builder{}

	// Содержимое формы
	formContent := strings.Builder{}
	formContent.WriteString(FormTitleStyle.Render("Add GitHub Account") + "\n\n")

	switch m.FormState {
	case models.NameInput:
		formContent.WriteString("Account Name:\n")
		formContent.WriteString(InputStyle.Render(m.NameInput.View()) + "\n\n")
		formContent.WriteString("Press Enter to continue, esc to cancel")
	case models.TokenInput:
		formContent.WriteString("GitHub Personal Access Token:\n")
		formContent.WriteString(InputStyle.Render(m.TokenInput.View()) + "\n\n")
		formContent.WriteString("Press Enter to save, esc to cancel")
	}

	// Сообщение
	if m.Message != "" {
		var style lipgloss.Style
		if m.MessageType == "success" {
			style = SuccessStyle
		} else {
			style = ErrorStyle
		}
		formContent.WriteString("\n\n" + style.Render(m.Message))
	}

	// Центрируем всю форму по центру экрана
	centeredForm := lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		formContent.String(),
	)
	doc.WriteString(centeredForm)

	return AppStyle.Render(doc.String())
}
