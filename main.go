package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// Модель данных аккаунта
type Account struct {
	Name    string    `json:"name"`
	Token   string    `json:"token"`
	Created time.Time `json:"created"`
	Private bool      `json:"private"`
	Client  *github.Client
}

func (a Account) Title() string { return a.Name }
func (a Account) Description() string {
	private := "Public"
	if a.Private {
		private = "Private"
	}
	return fmt.Sprintf("Created: %s • %s", a.Created.Format("2006-01-02"), private)
}
func (a Account) FilterValue() string { return a.Name }

// Модель данных репозитория
type Repository struct {
	Name      string
	Desc      string
	Stars     int
	Forks     int
	Language  string
	UpdatedAt time.Time
	IsPrivate bool
	SSHURL    string
	CloneURL  string
	Owner     string
}

func (r Repository) Title() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

func (r Repository) Description() string {
	desc := r.Desc
	if desc == "" {
		desc = "No description"
	}
	private := "Public"
	if r.IsPrivate {
		private = "Private"
	}
	return fmt.Sprintf("%s • %s • ⭐%d • 🍴%d • %s • Updated: %s",
		desc, private, r.Stars, r.Forks, r.Language, r.UpdatedAt.Format("2006-01-02"))
}

func (r Repository) FilterValue() string { return r.Name }

// Сообщения для обновления UI
type reposLoadedMsg struct {
	repos []Repository
	err   error
}

type cloneMsg struct {
	repo    Repository
	success bool
	err     error
	path    string
}

// Стили
var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	accountItemStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Margin(0, 1)

	activeAccountStyle = accountItemStyle.Copy().
				Foreground(lipgloss.Color("#25A065")).
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)

	addAccountStyle = accountItemStyle.Copy().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	activeAddAccountStyle = addAccountStyle.Copy().
				Foreground(lipgloss.Color("#25A065")).
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1)

	inputStyle = lipgloss.NewStyle().Width(40).BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	formTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#25A065")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#25A065"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

)

// Клавиши навигации
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Quit    key.Binding
	Submit  key.Binding
	Refresh key.Binding
	Clone   key.Binding
	Back    key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh repos"),
	),
	Clone: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clone repo"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc/backspace", "back"),
	),
}

// Состояния формы добавления аккаунта
const (
	nameInput = iota
	tokenInput
)

// Состояния приложения
const (
	stateAccounts = iota
	stateRepos
	stateAddingAccount
)

// Основная модель приложения
type model struct {
	accounts        []Account
	selectedAccount int // Индекс выбранного аккаунта
	accountsList    []string
	list            list.Model
	keys            keyMap
	width           int
	height          int
	state           int
	formState       int
	nameInput       textinput.Model
	tokenInput      textinput.Model
	configFile      string
	repos           []Repository
	selectedAccountPtr *Account
	spinner         spinner.Model
	loading         bool
	message         string
	messageType     string // "success" or "error"
}

// Файл для сохранения аккаунтов
func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".github_manager.json")
}

// Загрузка аккаунтов из файла
func loadAccounts(configFile string) ([]Account, error) {
	data, err := os.ReadFile(configFile)
	if os.IsNotExist(err) {
		return []Account{}, nil
	}
	if err != nil {
		return nil, err
	}
	var accounts []Account
	err = json.Unmarshal(data, &accounts)

	// Восстанавливаем клиенты GitHub
	for i := range accounts {
		if accounts[i].Token != "" {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: accounts[i].Token},
			)
			tc := oauth2.NewClient(context.Background(), ts)
			accounts[i].Client = github.NewClient(tc)
		}
	}

	return accounts, err
}

// Сохранение аккаунтов в файл
func saveAccounts(configFile string, accounts []Account) error {
	// Очищаем клиенты перед сохранением
	saveAccounts := make([]Account, len(accounts))
	for i, acc := range accounts {
		saveAccounts[i] = Account{
			Name:    acc.Name,
			Token:   acc.Token,
			Created: acc.Created,
			Private: acc.Private,
		}
	}

	data, err := json.MarshalIndent(saveAccounts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// Загрузка репозиториев
func loadRepos(account *Account) tea.Cmd {
	return func() tea.Msg {
		if account.Client == nil {
			return reposLoadedMsg{err: fmt.Errorf("GitHub client not initialized")}
		}

		var allRepos []*github.Repository
		opt := &github.RepositoryListOptions{
			Type:        "all",
			ListOptions: github.ListOptions{PerPage: 100},
		}

		for {
			repos, resp, err := account.Client.Repositories.List(context.Background(), "", opt)
			if err != nil {
				return reposLoadedMsg{err: err}
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}

		var convertedRepos []Repository
		for _, repo := range allRepos {
			language := ""
			if repo.Language != nil {
				language = *repo.Language
			}

			description := ""
			if repo.Description != nil {
				description = *repo.Description
			}

			owner := ""
			if repo.Owner != nil && repo.Owner.Login != nil {
				owner = *repo.Owner.Login
			}

			sshURL := ""
			if repo.SSHURL != nil {
				sshURL = *repo.SSHURL
			}

			cloneURL := ""
			if repo.CloneURL != nil {
				cloneURL = *repo.CloneURL
			}

			updatedAt := time.Now()
			if repo.UpdatedAt != nil {
				updatedAt = repo.UpdatedAt.Time
			}

			convertedRepos = append(convertedRepos, Repository{
				Name:      repo.GetName(),
				Desc:      description,
				Stars:     repo.GetStargazersCount(),
				Forks:     repo.GetForksCount(),
				Language:  language,
				UpdatedAt: updatedAt,
				IsPrivate: repo.GetPrivate(),
				SSHURL:    sshURL,
				CloneURL:  cloneURL,
				Owner:     owner,
			})
		}

		return reposLoadedMsg{repos: convertedRepos}
	}
}

// Клонирование репозитория
func cloneRepo(repo Repository, token string) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return cloneMsg{repo: repo, success: false, err: fmt.Errorf("failed to get home directory: %v", err)}
		}

		// Создаем путь: ~/develop/owner/repo-name
		devDir := filepath.Join(home, "develop")
		if err := os.MkdirAll(devDir, 0755); err != nil {
			return cloneMsg{repo: repo, success: false, err: fmt.Errorf("failed to create develop directory: %v", err)}
		}

		repoDir := filepath.Join(devDir, repo.Name)

		// Проверяем, существует ли репозиторий
		if _, err := os.Stat(repoDir); err == nil {
			return cloneMsg{
				repo:    repo,
				success: false,
				err:     fmt.Errorf("repository already exists at %s", repoDir),
				path:    repoDir,
			}
		}

		// Создаем URL с токеном для аутентификации
		cloneURL := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git",
			"oauth2",
			token,
			repo.Owner,
			repo.Name)

		// Клонируем репозиторий
		cmd := exec.Command("git", "clone", cloneURL, repoDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return cloneMsg{
				repo:    repo,
				success: false,
				err:     fmt.Errorf("git clone failed: %v, output: %s", err, string(output)),
				path:    repoDir,
			}
		}

		return cloneMsg{
			repo:    repo,
			success: true,
			path:    repoDir,
		}
	}
}

// Инициализация начального состояния
func initialModel() model {
	configFile := getConfigPath()
	accounts, err := loadAccounts(configFile)
	if err != nil {
		fmt.Printf("Error loading accounts: %v\n", err)
		accounts = []Account{}
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
	l.Styles.Title = titleStyle
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

	return model{
		accounts:        accounts,
		selectedAccount: 0,
		accountsList:    accountsList,
		list:            l,
		keys:            keys,
		configFile:      configFile,
		nameInput:       nameInput,
		tokenInput:      tokenInput,
		state:           stateAccounts,
		spinner:         s,
	}
}

// Инициализация программы
func (m model) Init() tea.Cmd {
	return nil
}

// Обновление состояния
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-appStyle.GetHorizontalFrameSize(), msg.Height-10)
		inputStyle = inputStyle.Width(min(40, msg.Width-20))

	case tea.KeyMsg:
		switch m.state {
		case stateAccounts:
			return m.updateAccountsState(msg)
		case stateRepos:
			return m.updateReposState(msg)
		case stateAddingAccount:
			return m.updateAddingAccountState(msg)
		}

	case reposLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading repositories: %v", msg.err)
			m.messageType = "error"
		} else {
			m.repos = msg.repos
			// Обновляем список
			items := make([]list.Item, len(m.repos))
			for i, repo := range m.repos {
				items[i] = repo
			}
			m.list.SetItems(items)
			m.message = fmt.Sprintf("Loaded %d repositories", len(m.repos))
			m.messageType = "success"
		}

	case cloneMsg:
		m.loading = false
		if msg.success {
			m.message = fmt.Sprintf("✅ Successfully cloned %s/%s\n📁 Path: %s",
				msg.repo.Owner, msg.repo.Name, msg.path)
			m.messageType = "success"

			// Проверяем, действительно ли репозиторий создан
			if _, err := os.Stat(msg.path); os.IsNotExist(err) {
				m.message = fmt.Sprintf("❌ Repository cloned but not found at expected path: %s", msg.path)
				m.messageType = "error"
			}
		} else {
			m.message = fmt.Sprintf("❌ Error cloning repository: %v", msg.err)
			m.messageType = "error"
		}
	}

	if m.loading {
		var spinCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinCmd)
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) updateAccountsState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.selectedAccount = max(m.selectedAccount-1, 0)
	case key.Matches(msg, m.keys.Down):
		m.selectedAccount = min(m.selectedAccount+1, len(m.accountsList)-1)
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Submit):
		if m.selectedAccount == len(m.accountsList)-1 {
			// Переход к добавлению аккаунта
			m.state = stateAddingAccount
			m.formState = nameInput
			m.nameInput.Focus()
		} else if m.selectedAccount < len(m.accounts) {
			// Загрузка репозиториев выбранного аккаунта
			m.selectedAccountPtr = &m.accounts[m.selectedAccount]
			m.state = stateRepos
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, loadRepos(m.selectedAccountPtr))
		}
	}
	return m, nil
}

func (m model) updateReposState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.state = stateAccounts
		m.message = ""
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, loadRepos(m.selectedAccountPtr))
	case key.Matches(msg, m.keys.Clone):
		if selectedItem := m.list.SelectedItem(); selectedItem != nil {
			if repo, ok := selectedItem.(Repository); ok {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, cloneRepo(repo, m.selectedAccountPtr.Token))
			}
		}
	default:
		m.list, _ = m.list.Update(msg)
	}
	return m, nil
}

func (m model) updateAddingAccountState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.state = stateAccounts
		m.nameInput.Reset()
		m.tokenInput.Reset()
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Submit):
		if m.formState == tokenInput {
			// Создаем новый аккаунт
			if m.nameInput.Value() != "" && m.tokenInput.Value() != "" {
				ts := oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: m.tokenInput.Value()},
				)
				tc := oauth2.NewClient(context.Background(), ts)

				newAccount := Account{
					Name:    m.nameInput.Value(),
					Token:   m.tokenInput.Value(),
					Created: time.Now(),
					Private: true,
					Client:  github.NewClient(tc),
				}

				// Добавляем аккаунт и сохраняем
				m.accounts = append(m.accounts, newAccount)
				if err := saveAccounts(m.configFile, m.accounts); err != nil {
					m.message = fmt.Sprintf("Error saving account: %v", err)
					m.messageType = "error"
				}

				// Обновляем список аккаунтов
				m.accountsList = []string{}
				for _, acc := range m.accounts {
					m.accountsList = append(m.accountsList, acc.Name)
				}
				m.accountsList = append(m.accountsList, "+ Add Account")

				// Переходим на новый аккаунт
				m.selectedAccount = len(m.accounts) - 1
				m.state = stateAccounts

				// Сбрасываем поля ввода
				m.nameInput.Reset()
				m.tokenInput.Reset()
				m.message = "Account added successfully"
				m.messageType = "success"
			}
		} else {
			// Переход к следующему полю
			m.formState = tokenInput
			m.nameInput.Blur()
			m.tokenInput.Focus()
		}
	default:
		// Обновляем активное поле ввода
		switch m.formState {
		case nameInput:
			m.nameInput, _ = m.nameInput.Update(msg)
		case tokenInput:
			m.tokenInput, _ = m.tokenInput.Update(msg)
		}
	}
	return m, nil
}

// Отображение интерфейса
func (m model) View() string {
	switch m.state {
	case stateAddingAccount:
		return m.renderAddAccountScreen()
	case stateRepos:
		return m.renderReposScreen()
	default:
		return m.renderAccountsScreen()
	}
}

// Рендер экрана аккаунтов
func (m model) renderAccountsScreen() string {
    doc := strings.Builder{}

    // Заголовок - центрируем по всей ширине
    title := titleStyle.Render("GitHub Account Manager")
    centeredTitle := lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, title)
    doc.WriteString(centeredTitle + "\n\n")

    // Список аккаунтов
    var accountItems []string
    for i, accountName := range m.accountsList {
        if i == m.selectedAccount {
            if i == len(m.accountsList)-1 {
                accountItems = append(accountItems, activeAddAccountStyle.Render(accountName))
            } else {
                accountItems = append(accountItems, activeAccountStyle.Render(accountName))
            }
        } else {
            if i == len(m.accountsList)-1 {
                accountItems = append(accountItems, addAccountStyle.Render(accountName))
            } else {
                accountItems = append(accountItems, accountItemStyle.Render(accountName))
            }
        }
    }

    // Объединяем аккаунты вертикально
    accountsView := lipgloss.JoinVertical(lipgloss.Center, accountItems...)
    
    // Центрируем весь блок по горизонтали и вертикали
    centeredAccounts := lipgloss.Place(
        m.width, 
        m.height-10, // Оставляем место для заголовка и инструкций
        lipgloss.Center, 
        lipgloss.Center, 
        accountsView,
    )
    doc.WriteString(centeredAccounts + "\n\n")

    // Сообщение
    if m.message != "" {
        var style lipgloss.Style
        if m.messageType == "success" {
            style = successStyle
        } else {
            style = errorStyle
        }
        centeredMessage := lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, style.Render(m.message))
        doc.WriteString(centeredMessage + "\n\n")
    }

    // Инструкции
    instructions := lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Italic(true).
        Render("Use ↑/↓ to navigate, Enter to select, q to quit")
    
    centeredInstructions := lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, instructions)
    doc.WriteString(centeredInstructions)

    return appStyle.Render(doc.String())
}

// Рендер экрана репозиториев
func (m model) renderReposScreen() string {
	doc := strings.Builder{}
	
	doc.WriteString(fmt.Sprintf("Account: %s\n", m.selectedAccountPtr.Name))
	
	// Показываем путь develop directory
	devPath, exists, err := checkDevelopDir()
	if err == nil {
		status := "✅"
		if !exists {
			status = "❌"
		}
		doc.WriteString(fmt.Sprintf("Develop directory: %s %s\n\n", devPath, status))
	}
	
	if m.loading {
		doc.WriteString(fmt.Sprintf("%s Loading repositories...\n\n", m.spinner.View()))
	} else {
		doc.WriteString(m.list.View() + "\n\n")
	}
	
	// Сообщение
	if m.message != "" {
		var style lipgloss.Style
		if m.messageType == "success" {
			style = successStyle
		} else {
			style = errorStyle
		}
		doc.WriteString(style.Render(m.message) + "\n\n")
	}
	
	doc.WriteString("Press c to clone, r to refresh, esc to back, q to quit")
	
	return appStyle.Render(doc.String())
}

// Рендер экрана добавления аккаунта
func (m model) renderAddAccountScreen() string {
    doc := strings.Builder{}

    // Содержимое формы
    formContent := strings.Builder{}
    formContent.WriteString(formTitleStyle.Render("Add GitHub Account") + "\n\n")

    switch m.formState {
    case nameInput:
        formContent.WriteString("Account Name:\n")
        formContent.WriteString(inputStyle.Render(m.nameInput.View()) + "\n\n")
        formContent.WriteString("Press Enter to continue, esc to cancel")
    case tokenInput:
        formContent.WriteString("GitHub Personal Access Token:\n")
        formContent.WriteString(inputStyle.Render(m.tokenInput.View()) + "\n\n")
        formContent.WriteString("Press Enter to save, esc to cancel")
    }

    // Сообщение
    if m.message != "" {
        var style lipgloss.Style
        if m.messageType == "success" {
            style = successStyle
        } else {
            style = errorStyle
        }
        formContent.WriteString("\n\n" + style.Render(m.message))
    }

    // Центрируем всю форму по центру экрана
    centeredForm := lipgloss.Place(
        m.width, 
        m.height, 
        lipgloss.Center, 
        lipgloss.Center, 
        formContent.String(),
    )
    doc.WriteString(centeredForm)

    return appStyle.Render(doc.String())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Получение пути к develop директории
func getDevelopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "develop"), nil
}

// Проверка существования develop директории
func checkDevelopDir() (string, bool, error) {
	devPath, err := getDevelopPath()
	if err != nil {
		return "", false, err
	}
	
	if _, err := os.Stat(devPath); os.IsNotExist(err) {
		return devPath, false, nil
	}
	
	return devPath, true, nil
}
