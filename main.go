package main

import (
	"context"
	"encoding/json"
	"fmt"
	botservice "go_mod/bot"
	"go_mod/config"
	"log"

	asynq "github.com/hibiken/asynq"
	redis "github.com/redis/go-redis/v9"

	"github.com/fatih/color"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if err := godotenv.Load(); err != nil {
		log.Println(red("No .env file found, using environment variables"))
	}

	cfg := config.LoadConfig()

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal(red("Failed to initialize Telegram Bot API: %s", err))
		return
	}

	bot.Debug = true

	log.Println(green("Authorized on account %s", bot.Self.UserName))

	clientOptions := options.Client().ApplyURI(cfg.MongoDBURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(red(err))
	}

	fmt.Println(green("Connected to MongoDB!"))

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURI,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	pong, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(red("Failed to connect to Redis: %s", err))
	}
	fmt.Println(green("Connected to Redis!\n", pong))

	clientAsynq := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURI, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	defer clientAsynq.Close()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisURI, Password: cfg.RedisPassword, DB: cfg.RedisDB},
		asynq.Config{
			Concurrency: 10,
		})

	collection := client.Database(cfg.MongoDBDatabase).Collection("tasks")
	botService := botservice.NewBotService(bot, collection, rdb, clientAsynq)

	command, err := botService.GetCommandState(bot.Self.ID)
	if err != nil {
		log.Println(red("Ошибка при получении состояния команды:", err))
	} else if command != "" {
		log.Println(green("Текущая команда пользователя:", command))
	}

	taskHandler := func(ctx context.Context, t *asynq.Task) error {
		var taskInfo botservice.Task
		if err := json.Unmarshal(t.Payload(), &taskInfo); err != nil {
			return fmt.Errorf("json.Unmarshal failed: %v", err)
		}
		return nil
	}

	mux := asynq.NewServeMux()
	mux.HandleFunc("reminder:send", taskHandler)

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatal(red("Asynq server failed to start: %v", err))
		}
	}()

	go botService.StartReminder(cfg.ReminderIntervalMinutes, clientAsynq)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			botService.HandleCommand(update.Message, clientAsynq)
		}
	}
}
