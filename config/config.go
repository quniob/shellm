package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ApiKey           string `mapstructure:"api_key"`
	ApiBaseUrl       string `mapstructure:"api_base_url"`
	ApiModel         string `mapstructure:"api_model"`
	InventoryPath    string `mapstructure:"inventory_path"`
	SecretsPath      string `mapstructure:"secrets_path"`
	LLMMaxIterations int    `mapstructure:"llm_max_iterations"`
	LLMTimeOut       int    `mapstructure:"llm_timeout"`
}

func LoadConfig() (*Config, error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("shellm")

	viper.SetDefault("api_base_url", "https://openrouter.ai/api/v1")
	viper.SetDefault("api_model", "google/gemini-2.5-pro")
	viper.SetDefault("inventory_path", "./inventory")
	viper.SetDefault("secrets_path", "./secrets")
	viper.SetDefault("llm_max_iterations", 10)
	viper.SetDefault("llm_timeout", 60)

	viper.BindEnv("api_key")
	viper.BindEnv("api_base_url")
	viper.BindEnv("api_model")
	viper.BindEnv("inventory_path")
	viper.BindEnv("secrets_path")
	viper.BindEnv("llm_max_iterations")
	viper.BindEnv("llm_timeout")

	viper.AutomaticEnv()
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
