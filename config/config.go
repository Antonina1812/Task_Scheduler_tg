package config

import (
	"os"
)

type Config struct {
	TelegramToken           string
	MongoDBURI              string
	MongoDBDatabase         string
	RedisURI                string
	RedisPassword           string
	RedisDB                 int
	ReminderIntervalMinutes int
}

func LoadConfig() Config {
	return Config{
		TelegramToken:           os.Getenv("TELEGRAM_TOKEN"),
		MongoDBURI:              os.Getenv("MONGODB_URI"),
		MongoDBDatabase:         os.Getenv("MONGODB_DATABASE"),
		ReminderIntervalMinutes: 1,
	}

}
