package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var C config

type config struct {
	Http http
}

type http struct {
	Host string
	Port int64
}

func MustLoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/ect/hot_update/")
	viper.AddConfigPath("$HOME/.hot_update")
	viper.AddConfigPath("./config")
	err := viper.ReadInConfig()
	if err != nil {
		return errors.Wrap(err, "read config")
	}
	viper.WatchConfig()
	err = viper.Unmarshal(&C)
	if err != nil {
		return errors.Wrap(err, "unmarshal config")
	}
	return nil
}
