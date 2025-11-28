package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

/*
	In local development

	# Use .env file in local dev
	github.com/joho/godotenv

	# run without config file
	go run .

	# run with value get from config file
	go run . --config config/config.example.yaml

	# override value by env var (after load from file)
	SERVER_PORT=8081 DATABASE_HOST=local.db.com go run . --config config/config.example.yaml

	In k8s, not mount config file, use ConfigMap & Secret to inject the config via
	environment variables
*/

// Config holds the configuration settings for the application.
type Config struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`

	Database struct {
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
	} `mapstructure:"database"`

	SMTP struct {
		Host      string `mapstructure:"host"`
		Port      int    `mapstructure:"port"`
		User      string `mapstructure:"user"`
		Password  string `mapstructure:"password"`
		JWTSecret string `mapstructure:"jwtSecret"`
	} `mapstructure:"smtp"`

	Redis struct {
		Address  string `mapstructure:"address"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
}

func loadConfig(configPath string) (*Config, error) {
	// create new viper instance
	v := viper.New()

	// set default (lowest priority)
	setDefaults(v)

	// read from file (if exists)
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			// if file not exist, use env var or default
			log.Printf("Warning: cannot read config file: %v\n", err)
		} else {
			log.Printf("Using config file: %s", v.ConfigFileUsed())
		}
	}

	// read from environment variables (higher priority than file), viper should
	// automatically find env var matching with config struct. Ex: `database.host`
	// matching by `DATABASE_HOST`
	v.AutomaticEnv()
	// replace '.' by '_' in env var for consistency
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
}

func RunLocalEnvExample(skip bool) {
	if skip {
		return
	}

	// create --config flag to add config path
	configPath := pflag.StringP("config", "c", "", "Path to config file")
	pflag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	// use the config struct within application
	fmt.Printf("App config loaded:\nServer: Host=%s, Port=%d\nDatabase: Host=%s, Port=%d, User=%s, Password=%s, Name=%s\n",
		cfg.Server.Host, cfg.Server.Port, cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	)
}

func LoadLocalConfig() (*Config, error) {
	return loadConfig("config/config.yaml")
}
