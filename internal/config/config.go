package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type EmailConfig struct {
	SendEmail  string   `mapstructure:"send_email"`
	SMTPHost   string   `mapstructure:"smtp_host"`
	SMTPPort   int      `mapstructure:"smtp_port"`
	Subscriber []string `mapstructure:"send_email_to"`
}

type ProxyConfig struct {
	HandshakeTimeout int `mapstructure:"handshake_timeout"`
	MsgTimeout       int `mapstructure:"msg_timeout"`
	MaxRetries       int `mapstructure:"max_retries"`
	RetryInterval    int `mapstructure:"retry_interval"`
}

type Config struct {
	Email EmailConfig `mapstructure:"email"`
	Proxy ProxyConfig `mapstructure:"proxy"`
}

func LoadConfig(cfgFile string) Config {
	viper.SetConfigType("json")
	// Provide sensible defaults so the binary works even without a config file.
	viper.SetDefault("email.send_email", "")
	viper.SetDefault("email.smtp_host", "")
	viper.SetDefault("email.smtp_port", 0)
	viper.SetDefault("email.send_email_to", []string{})

	viper.SetDefault("proxy.handshake_timeout", 5000)
	viper.SetDefault("proxy.msg_timeout", 5000)
	viper.SetDefault("proxy.max_retries", 3)
	viper.SetDefault("proxy.retry_interval", 200)

	configPath := cfgFile

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve home directory: %v\n", err)
			return Config{}
		}
		configPath = filepath.Join(home, ".config", "cogmoteGO", "config.json")
	}

	viper.SetConfigFile(configPath)

	// Make sure the containing folder exists before reading/writing configs.
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create config directory: %v\n", err)
		return Config{}
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		// Generate a new config file populated with the defaults on first run.
		if writeErr := viper.WriteConfigAs(configPath); writeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to create config: %v\n", writeErr)
			return Config{}
		}
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check config file: %v\n", err)
		return Config{}
	} else if err := viper.ReadInConfig(); err != nil {
		// Config exists but cannot be read; report the error and exit early.
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		return Config{}
	}

	var cfg Config
	// Deserialize the config file into the strongly typed struct used elsewhere.
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse config: %v\n", err)
		return Config{}
	}

	return cfg
}
