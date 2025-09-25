package utils

import (
	"os"
	"path/filepath"
)

// GetDevelopPath возвращает путь к директории develop
func GetDevelopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "develop"), nil
}

// CheckDevelopDir проверяет существование директории develop
func CheckDevelopDir() (string, bool, error) {
	devPath, err := GetDevelopPath()
	if err != nil {
		return "", false, err
	}

	if _, err := os.Stat(devPath); os.IsNotExist(err) {
		return devPath, false, nil
	}

	return devPath, true, nil
}
