package models

// Состояния формы добавления аккаунта
const (
	NameInput = iota
	TokenInput
)

// Состояния приложения
const (
	StateAccounts = iota
	StateRepos
	StateAddingAccount
)
