package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func GetCredentials(key string) (value string, err error) {
	home, _ := os.UserHomeDir()

	viper.SetConfigFile(filepath.Join(home, "config.yaml"))

	if err := viper.ReadInConfig(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if !viper.IsSet(key) {
		return "", fmt.Errorf("value of %s is not set", key)
	}

	return viper.GetString(key), nil
}
