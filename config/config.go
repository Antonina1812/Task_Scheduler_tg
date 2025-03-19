package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
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
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	return Config{
		TelegramToken:           os.Getenv("TELEGRAM_TOKEN"),
		MongoDBURI:              os.Getenv("MONGODB_URI"),
		MongoDBDatabase:         os.Getenv("MONGODB_DATABASE"),
		RedisURI:                os.Getenv("REDIS_URI"),
		RedisPassword:           os.Getenv("REDIS_PASSWORD"),
		ReminderIntervalMinutes: 1,
		RedisDB:                 0,
	}

}
