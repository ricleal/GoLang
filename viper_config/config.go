package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// server:
//   host: localhost
//   port: 8080
//   timeout: 10s

type Config struct {
	Server struct {
		Host    string
		Port    int
		Timeout time.Duration
	}
}

func Setup() *Config {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	viper.AutomaticEnv()          // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %w", err))
	}

	return &config
}
