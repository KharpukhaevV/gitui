package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/KharpukhaevV/gitui/models"
	"github.com/KharpukhaevV/gitui/utils"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// Manager управляет конфигурацией аккаунтов
type Manager struct {
	configFile string
}

// NewManager создает новый менеджер конфигурации
func NewManager() *Manager {
	return &Manager{
		configFile: getConfigPath(),
	}
}

// getConfigPath возвращает путь к файлу конфигурации
func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".github_manager.json")
}

// LoadAccounts загружает аккаунты из файла
func (m *Manager) LoadAccounts() ([]models.Account, error) {
	data, err := os.ReadFile(m.configFile)
	if os.IsNotExist(err) {
		return []models.Account{}, nil
	}
	if err != nil {
		return nil, err
	}

	var accounts []models.Account
	err = json.Unmarshal(data, &accounts)
	if err != nil {
		return nil, err
	}

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

	return accounts, nil
}

// SaveAccounts сохраняет аккаунты в файл
func (m *Manager) SaveAccounts(accounts []models.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	// Очищаем клиенты перед сохранением
	saveAccounts := make([]models.Account, len(accounts))
	for i, acc := range accounts {
		saveAccounts[i] = models.Account{
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
	return os.WriteFile(m.configFile, data, utils.DefaultFileMode)
}
