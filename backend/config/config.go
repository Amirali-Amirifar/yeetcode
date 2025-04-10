package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

const configPath = "config/config.dev.yaml"

var (
	config    Config
	loadOnce  sync.Once
	configErr error
)

func GetConfig() Config {
	loadOnce.Do(func() {
		cfg, err := getViperConfig()
		if err != nil {
			configErr = err
			return
		}
		config = *cfg
	})

	if configErr != nil {
		panic(fmt.Sprintf("Couldn't get config: %s", configErr))
	}

	return config
}

func getViperConfig() (*Config, error) {
	var config Config
	viper.SetConfigName(configPath)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
