package config

import "github.com/spf13/viper"

type Config struct {
	Server struct {
		ListenAddr       string `mapstructure:"listen_addr"`
		ShutdownTimeoutS int    `mapstructure:"shutdown_timeout_s"`
	} `mapstructure:"server"`

	Bus struct {
		TopicBufferSize     int `mapstructure:"topic_buffer_size"`
		SubscriberQueueSize int `mapstructure:"subscriber_queue_size"`
	} `mapstructure:"bus"`
}

func InitConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
