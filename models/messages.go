package models

// ReposLoadedMsg сообщение о загрузке репозиториев
type ReposLoadedMsg struct {
	Repos []Repository
	Err   error
}

// CloneMsg сообщение о клонировании репозитория
type CloneMsg struct {
	Repo    Repository
	Success bool
	Err     error
	Path    string
}
