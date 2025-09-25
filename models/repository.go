package models

import (
	"fmt"
	"time"
)

// Repository представляет репозиторий GitHub
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

// Title возвращает название репозитория для отображения в списке
func (r Repository) Title() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// Description возвращает описание репозитория для отображения в списке
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

// FilterValue возвращает значение для фильтрации
func (r Repository) FilterValue() string {
	return r.Name
}
