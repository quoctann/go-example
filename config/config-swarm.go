package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Example load config for docker swarm
func LoadSwarmConfig(configPath string) (*Config, error) {
	// same as load local
	v := viper.New()
	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			// if file not exist, use env var or default
			log.Printf("Warning: cannot read config file: %v\n", err)
		} else {
			log.Printf("Using config file: %s", v.ConfigFileUsed())
		}
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// (DIFFERENT) process docker swarm secrets mounting as file (usually mount
	// into /run/secrets), example: if env var DATABASE_PASSWORD, try read from
	// file
	if !v.IsSet("database.password") {
		secretPath := "/run/secrets/db_password_file" // filename in docker compose
		if content, err := os.ReadFile(secretPath); err == nil {
			v.Set("database.password", strings.TrimSpace(string(content)))
			log.Printf("Loaded database.password form docker secret file: %s\n", secretPath)
		}
	} // do the same for other secret value like db user,...

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct, %v", err)
	}
	return &cfg, nil
}
