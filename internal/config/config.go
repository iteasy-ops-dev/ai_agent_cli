package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/yourusername/syseng-agent/pkg/types"
)

func Load() (*types.Config, error) {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("database.type", "memory")
	viper.SetDefault("database.path", "")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("agent.timeout", 30)
	
	viper.SetEnvPrefix("SYSENG_AGENT")
	viper.AutomaticEnv()
	
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	viper.AddConfigPath(".")
	viper.AddConfigPath(filepath.Join(home, ".syseng-agent"))
	viper.AddConfigPath("/etc/syseng-agent")
	
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}
	
	var config types.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

func Save(config *types.Config) error {
	viper.Set("server", config.Server)
	viper.Set("database", config.Database)
	viper.Set("logging", config.Logging)
	viper.Set("agent", config.Agent)
	
	return viper.WriteConfig()
}

func GetConfigPath() string {
	return viper.ConfigFileUsed()
}