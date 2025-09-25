package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/KharpukhaevV/gitui/config"
	githubClient "github.com/KharpukhaevV/gitui/github"
	"github.com/KharpukhaevV/gitui/models"
	"github.com/KharpukhaevV/gitui/utils"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// AppModel основная модель приложения
type AppModel struct {
	Accounts           []models.Account
	SelectedAccount    int
	AccountsList       []string
	List               list.Model
	Keys               KeyMap
	Width              int
	Height             int
	State              int
	FormState          int
	NameInput          textinput.Model
	TokenInput         textinput.Model
	ConfigManager      *config.Manager
	GitHubClient       *githubClient.Client
	Repos              []models.Repository
	SelectedAccountPtr *models.Account
	Spinner            spinner.Model
	Loading            bool
	Message            string
	MessageType        string // "success" or "error"
}

// NewAppModel создает новую модель приложения
func NewAppModel() *AppModel {
	configManager := config.NewManager()
	accounts, err := configManager.LoadAccounts()
	if err != nil {
		fmt.Printf("Error loading accounts: %v\n", err)
		accounts = []models.Account{}
	}

	// Создаем список аккаунтов
	accountsList := []string{}
	for _, acc := range accounts {
		accountsList = append(accountsList, acc.Name)
	}
	accountsList = append(accountsList, "+ Add Account")

	// Инициализация списка
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Repositories"
	l.Styles.Title = TitleStyle
	l.SetShowHelp(false)

	// Инициализация полей ввода
	nameInput := textinput.New()
	nameInput.Placeholder = "Account Name"
	nameInput.Focus()

	tokenInput := textinput.New()
	tokenInput.Placeholder = "GitHub Personal Access Token"
	tokenInput.EchoMode = textinput.EchoPassword

	// Инициализация спиннера
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &AppModel{
		Accounts:        accounts,
		SelectedAccount: 0,
		AccountsList:    accountsList,
		List:            l,
		Keys:            DefaultKeys(),
		ConfigManager:   configManager,
		GitHubClient:    githubClient.NewClient(),
		NameInput:       nameInput,
		TokenInput:      tokenInput,
		State:           models.StateAccounts,
		Spinner:         s,
	}
}

// Init инициализация программы
func (m *AppModel) Init() tea.Cmd {
	return nil
}

// Update обновление состояния
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.List.SetSize(msg.Width-AppStyle.GetHorizontalFrameSize(), msg.Height-10)
		InputStyle = InputStyle.Width(utils.Min(utils.DefaultInputWidth, msg.Width-utils.MinInputWidth))

	case tea.KeyMsg:
		switch m.State {
		case models.StateAccounts:
			return m.updateAccountsState(msg)
		case models.StateRepos:
			return m.updateReposState(msg)
		case models.StateAddingAccount:
			return m.updateAddingAccountState(msg)
		}

	case models.ReposLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Message = fmt.Sprintf("Error loading repositories: %v", msg.Err)
			m.MessageType = "error"
		} else {
			m.Repos = msg.Repos
			// Обновляем список
			items := make([]list.Item, len(m.Repos))
			for i, repo := range m.Repos {
				items[i] = repo
			}
			m.List.SetItems(items)
			m.Message = fmt.Sprintf("Loaded %d repositories", len(m.Repos))
			m.MessageType = "success"
		}

	case models.CloneMsg:
		m.Loading = false
		if msg.Success {
			m.Message = fmt.Sprintf("✅ Successfully cloned %s/%s\n📁 Path: %s",
				msg.Repo.Owner, msg.Repo.Name, msg.Path)
			m.MessageType = "success"
		} else {
			m.Message = fmt.Sprintf("❌ Error cloning repository: %v", msg.Err)
			m.MessageType = "error"
		}
	}

	if m.Loading {
		var spinCmd tea.Cmd
		m.Spinner, spinCmd = m.Spinner.Update(msg)
		cmds = append(cmds, spinCmd)
	}

	m.List, cmd = m.List.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View отображение интерфейса
func (m *AppModel) View() string {
	switch m.State {
	case models.StateAddingAccount:
		return RenderAddAccountScreen(m)
	case models.StateRepos:
		return RenderReposScreen(m)
	default:
		return RenderAccountsScreen(m)
	}
}

// updateAccountsState обновление состояния выбора аккаунтов
func (m *AppModel) updateAccountsState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "up" || msg.String() == "k":
		m.SelectedAccount = utils.Max(m.SelectedAccount-1, 0)
	case msg.String() == "down" || msg.String() == "j":
		m.SelectedAccount = utils.Min(m.SelectedAccount+1, len(m.AccountsList)-1)
	case msg.String() == "ctrl+c" || msg.String() == "q":
		return m, tea.Quit
	case msg.String() == "enter":
		if m.SelectedAccount == len(m.AccountsList)-1 {
			// Переход к добавлению аккаунта
			m.State = models.StateAddingAccount
			m.FormState = models.NameInput
			m.NameInput.Focus()
		} else if m.SelectedAccount < len(m.Accounts) {
			// Загрузка репозиториев выбранного аккаунта
			m.SelectedAccountPtr = &m.Accounts[m.SelectedAccount]
			m.State = models.StateRepos
			m.Loading = true
			return m, tea.Batch(m.Spinner.Tick, m.GitHubClient.LoadRepos(m.SelectedAccountPtr))
		}
	}
	return m, nil
}

// updateReposState обновление состояния репозиториев
func (m *AppModel) updateReposState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc" || msg.String() == "backspace":
		m.State = models.StateAccounts
		m.Message = ""
	case msg.String() == "ctrl+c" || msg.String() == "q":
		return m, tea.Quit
	case msg.String() == "r":
		m.Loading = true
		return m, tea.Batch(m.Spinner.Tick, m.GitHubClient.LoadRepos(m.SelectedAccountPtr))
	case msg.String() == "c":
		if selectedItem := m.List.SelectedItem(); selectedItem != nil {
			if repo, ok := selectedItem.(models.Repository); ok {
				m.Loading = true
				return m, tea.Batch(m.Spinner.Tick, m.GitHubClient.CloneRepo(repo, m.SelectedAccountPtr.Token))
			}
		}
	default:
		m.List, _ = m.List.Update(msg)
	}
	return m, nil
}

// updateAddingAccountState обновление состояния добавления аккаунта
func (m *AppModel) updateAddingAccountState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc" || msg.String() == "backspace":
		m.State = models.StateAccounts
		m.NameInput.Reset()
		m.TokenInput.Reset()
	case msg.String() == "ctrl+c" || msg.String() == "q":
		return m, tea.Quit
	case msg.String() == "enter":
		if m.FormState == models.TokenInput {
			// Создаем новый аккаунт
			if m.NameInput.Value() != "" && m.TokenInput.Value() != "" {
				ts := oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: m.TokenInput.Value()},
				)
				tc := oauth2.NewClient(context.Background(), ts)

				newAccount := models.Account{
					Name:    m.NameInput.Value(),
					Token:   m.TokenInput.Value(),
					Created: time.Now(),
					Private: true,
					Client:  github.NewClient(tc),
				}

				// Добавляем аккаунт и сохраняем
				m.Accounts = append(m.Accounts, newAccount)
				if err := m.ConfigManager.SaveAccounts(m.Accounts); err != nil {
					m.Message = fmt.Sprintf("Error saving account: %v", err)
					m.MessageType = "error"
				}

				// Обновляем список аккаунтов
				m.AccountsList = []string{}
				for _, acc := range m.Accounts {
					m.AccountsList = append(m.AccountsList, acc.Name)
				}
				m.AccountsList = append(m.AccountsList, "+ Add Account")

				// Переходим на новый аккаунт
				m.SelectedAccount = len(m.Accounts) - 1
				m.State = models.StateAccounts

				// Сбрасываем поля ввода
				m.NameInput.Reset()
				m.TokenInput.Reset()
				m.Message = "Account added successfully"
				m.MessageType = "success"
			}
		} else {
			// Переход к следующему полю
			m.FormState = models.TokenInput
			m.NameInput.Blur()
			m.TokenInput.Focus()
		}
	default:
		// Обновляем активное поле ввода
		switch m.FormState {
		case models.NameInput:
			m.NameInput, _ = m.NameInput.Update(msg)
		case models.TokenInput:
			m.TokenInput, _ = m.TokenInput.Update(msg)
		}
	}
	return m, nil
}
