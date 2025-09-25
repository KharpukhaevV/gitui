package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/KharpukhaevV/gitui/models"
	"github.com/KharpukhaevV/gitui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/go-github/github"
)

// Client предоставляет методы для работы с GitHub API
type Client struct{}

// NewClient создает новый клиент GitHub
func NewClient() *Client {
	return &Client{}
}

// LoadRepos загружает репозитории для указанного аккаунта
func (c *Client) LoadRepos(account *models.Account) tea.Cmd {
	return func() tea.Msg {
		if account == nil {
			return models.ReposLoadedMsg{Err: fmt.Errorf("account is nil")}
		}
		if account.Client == nil {
			return models.ReposLoadedMsg{Err: fmt.Errorf("GitHub client not initialized")}
		}

		var allRepos []*github.Repository
		opt := &github.RepositoryListOptions{
			Type:        "all",
			ListOptions: github.ListOptions{PerPage: 100},
		}

		for {
			repos, resp, err := account.Client.Repositories.List(context.Background(), "", opt)
			if err != nil {
				return models.ReposLoadedMsg{Err: err}
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}

		var convertedRepos []models.Repository
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

			convertedRepos = append(convertedRepos, models.Repository{
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

		return models.ReposLoadedMsg{Repos: convertedRepos}
	}
}

// CloneRepo клонирует репозиторий
func (c *Client) CloneRepo(repo models.Repository, token string) tea.Cmd {
	return func() tea.Msg {
		if token == "" {
			return models.CloneMsg{Repo: repo, Success: false, Err: fmt.Errorf("token is empty")}
		}
		if repo.Owner == "" || repo.Name == "" {
			return models.CloneMsg{Repo: repo, Success: false, Err: fmt.Errorf("repository owner or name is empty")}
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return models.CloneMsg{Repo: repo, Success: false, Err: fmt.Errorf("failed to get home directory: %v", err)}
		}

		// Создаем путь: ~/develop/owner/repo-name
		devDir := filepath.Join(home, "develop")
		if err := os.MkdirAll(devDir, utils.DefaultDirMode); err != nil {
			return models.CloneMsg{Repo: repo, Success: false, Err: fmt.Errorf("failed to create develop directory: %v", err)}
		}

		repoDir := filepath.Join(devDir, repo.Name)

		// Проверяем, существует ли репозиторий
		if _, err := os.Stat(repoDir); err == nil {
			return models.CloneMsg{
				Repo:    repo,
				Success: false,
				Err:     fmt.Errorf("repository already exists at %s", repoDir),
				Path:    repoDir,
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
			return models.CloneMsg{
				Repo:    repo,
				Success: false,
				Err:     fmt.Errorf("git clone failed: %v, output: %s", err, string(output)),
				Path:    repoDir,
			}
		}

		return models.CloneMsg{
			Repo:    repo,
			Success: true,
			Path:    repoDir,
		}
	}
}
