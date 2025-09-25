package models

import (
	"fmt"
	"time"

	"github.com/google/go-github/github"
)

// Account представляет аккаунт GitHub
type Account struct {
	Name    string    `json:"name"`
	Token   string    `json:"token"`
	Created time.Time `json:"created"`
	Private bool      `json:"private"`
	Client  *github.Client
}

// Title возвращает название аккаунта для отображения в списке
func (a Account) Title() string {
	return a.Name
}

// Description возвращает описание аккаунта для отображения в списке
func (a Account) Description() string {
	private := "Public"
	if a.Private {
		private = "Private"
	}
	return fmt.Sprintf("Created: %s • %s", a.Created.Format("2006-01-02"), private)
}

// FilterValue возвращает значение для фильтрации
func (a Account) FilterValue() string {
	return a.Name
}
