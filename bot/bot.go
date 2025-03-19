package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	asynq "github.com/hibiken/asynq"

	redis "github.com/redis/go-redis/v9"

	"github.com/fatih/color"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

//TODO: подключить очередь задач
//TODO: добавить нейронку или альтернативу

//TODO: обернуть в докер контейнер
//TODO: добавить кнопки

type Task struct {
	ChatID         int64     `bson:"chat_id"`
	Description    string    `bson:"description"`
	CreatedAt      time.Time `bson:"created_at"`
	Deadline       time.Time `bson:"deadline"`
	Mark           bool      `bson:"mark"`
	ReminderExists bool      `bson:"reminder"`
}

type TaskStatistics struct {
	CompletedOnTime     int
	Overdue             int
	AverageDeadlineDays float64
}

type BotService struct {
	api      *tgbotapi.BotAPI
	db       *mongo.Collection
	rdb      *redis.Client
	redisCtx context.Context
}

func NewBotService(api *tgbotapi.BotAPI, db *mongo.Collection, redisClient *redis.Client) *BotService {
	return &BotService{
		api:      api,
		db:       db,
		rdb:      redisClient,
		redisCtx: context.Background(),
	}
}

func (bs *BotService) HandleCommand(message *tgbotapi.Message, client *asynq.Client) {
	chatID := message.Chat.ID
	command := message.Command()
	text := message.CommandArguments()

	switch command {
	case "start":
		bs.RunSettedCommand(chatID, "start")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "start" {
			bs.SendMessage(chatID, "Привет! Я бот, который поможет тебе управлять задачами. Используй /help для просмотра доступных команд.")
		}
	case "help":
		bs.RunSettedCommand(chatID, "help")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "help" {
			bs.SendMessage(chatID, "Доступные команды:\n/add <описание задачи> - добавить задачу\n/list - список задач\n/delete <дата в формате YYYY-MM-DD HH:MM> - удалить задачу\n/is_done <текст задачи> - отметить задачу, как выполненную\n/edit <старое описание задачи> | <новое описание задачи>\n/set_deadline <описание задачи> | <дата в формате YYYY-MM-DD HH:MM>\n/set_reminder <описание задачи> - установить напоминание\n/unset_reminder <описание задачи> - отменить напоминание\n/stats - просмотр статистики\n/help - помощь")
		}
	case "add":
		bs.RunSettedCommand(chatID, "add")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "add" {
			bs.AddTask(chatID, text, client)
		}
	case "set_deadline":
		bs.RunSettedCommand(chatID, "set_deadline")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "set_deadline" {
			bs.SetDeadline(chatID, text, client)
		}
	case "list":
		bs.RunSettedCommand(chatID, "list")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "list" {
			bs.ListTasks(chatID)
		}
	case "delete":
		bs.RunSettedCommand(chatID, "delete")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "delete" {
			bs.DeleteTask(chatID, text)
		}
	case "edit":
		bs.RunSettedCommand(chatID, "edit")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "edit" {
			bs.EditTask(chatID, text)
		}
	case "is_done":
		bs.RunSettedCommand(chatID, "is_done")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "is_done" {
			bs.IsDone(chatID, text)
		}
	case "set_reminder":
		bs.RunSettedCommand(chatID, "set_reminder")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "set_reminder" {
			bs.SetReminder(chatID, text, true, client)
		}
	case "unset_reminder":
		bs.RunSettedCommand(chatID, "unset_reminder")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "unset_reminder" {
			bs.SetReminder(chatID, text, false, client)
		}
	case "stats":
		bs.RunSettedCommand(chatID, "stats")
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
			return
		}
		if state == "stats" {
			bs.ShowStats(chatID)
		}
	case "":
		textWithoutCommand := message.Text
		state, err := bs.GetCommandState(chatID)
		if err != nil {
			log.Printf("Failed to get command state: %v", err)
			bs.SendMessage(chatID, "Произошла ошибка. Пожалуйста, попробуйте заново ввести команду.")
		}
		bs.ChooseMethod(chatID, state, textWithoutCommand, client)
	default:
		bs.SendMessage(chatID, "Неизвестная команда. Используйте /help для просмотра доступных команд.")
	}
}

func (bs *BotService) RunSettedCommand(chatID int64, command string) {
	red := color.New(color.FgRed).SprintFunc()
	err := bs.SetCommandState(chatID, command)
	if err != nil {
		log.Println(red("Failed to set command state: %v", err))
		bs.SendMessage(chatID, "Произошла ошибка при обработке команды. Попробуйте позже.")
		return
	}
}

func (bs *BotService) SetCommandState(userID int64, command string) error {
	red := color.New(color.FgRed).SprintFunc()
	key := fmt.Sprintf("user:%d:command", userID)
	err := bs.rdb.Set(bs.redisCtx, key, command, time.Minute*1).Err()
	if err != nil {
		log.Println(red("Failed to set command state: %v", err))
		return err
	}
	return nil
}

func (bs *BotService) GetCommandState(userID int64) (string, error) {

	key := fmt.Sprintf("user:%d:command", userID)
	cmd, err := bs.rdb.Get(bs.redisCtx, key).Result()

	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		log.Printf("Failed to get command state: %v", err)
		return "", err
	}
	return cmd, nil
}

func (bs *BotService) SendMessage(chatID int64, text string) {
	red := color.New(color.FgRed).SprintFunc()
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bs.api.Send(msg)
	if err != nil {
		log.Println(red("Failed to send message: %s", err))
	}
}

func (bs *BotService) ListTasks(chatID int64) {

	filter := bson.M{"chat_id": chatID}
	cursor, err := bs.db.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("Failed to list tasks: %s", err)
		bs.SendMessage(chatID, "Не удалось получить список задач.")
		return
	}
	defer cursor.Close(context.TODO())

	var tasks []Task
	if err := cursor.All(context.TODO(), &tasks); err != nil {
		log.Printf("Failed to decode tasks: %s", err)
		bs.SendMessage(chatID, "Не удалось получить список задач.")
		return
	}

	if len(tasks) == 0 {
		bs.SendMessage(chatID, "Список задач пуст.")
		return
	}

	message := "Список задач:\n"
	for i, task := range tasks {

		deadlineStr := "-"
		timeLeftStr := ""
		timeLeft := time.Until(task.Deadline)

		if !task.Deadline.IsZero() {
			deadlineStr = task.Deadline.Format("02 Jan 2006 15:04")

			if timeLeft > 0 {
				days := int(timeLeft.Hours()) / 24
				hours := int(timeLeft.Hours()) % 24
				minutes := int(timeLeft.Minutes()) % 60
				timeLeftStr = fmt.Sprintf(" (Осталось: %d дн. %d ч. %d мин.)", days, hours, minutes)
			} else {
				timeLeftStr = " (Просрочено)"
			}
		}

		if !task.Deadline.IsZero() {
			deadlineStr = task.Deadline.Format("02 Jan 2006 15:04")
		}

		if !task.Mark {
			message += fmt.Sprintf("%d. %s (Дедлайн: %s)🔥 %s\n", i+1, task.Description, deadlineStr, timeLeftStr)
		} else {
			message += fmt.Sprintf("%d. %s (Дедлайн: %s)✅ %s\n", i+1, task.Description, deadlineStr, timeLeftStr)
		}
	}
	bs.SendMessage(chatID, message)
}

func (bs *BotService) DeleteTask(chatID int64, text string) {
	if text == "" {
		bs.SendMessage(chatID, "Напишите дедлайн задачи в формате YYYY-MM-DD HH:MM.")
		return
	}

	taskDeadline, err := time.Parse("2006-01-02 15:04", text)
	if err != nil {
		log.Printf("Failed to decode tasks: %s", err)
		bs.SendMessage(chatID, "Неправильный формат.")
		return
	}

	filter := bson.M{"chat_id": chatID, "deadline": taskDeadline}

	result, err := bs.db.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Printf("Failed to delete task: %s", err)
		bs.SendMessage(chatID, "Не получилось удалить задачу.")
		return
	}

	if result.DeletedCount == 0 && text != "" {
		bs.SendMessage(chatID, "Задача не найдена.")
		return
	}

	bs.SendMessage(chatID, "Задача удалена!")
}

func (bs *BotService) AddTask(chatID int64, description string, client *asynq.Client) {
	if description == "" {
		bs.SendMessage(chatID, "Пожалуйста, укажите описание задачи.")
		return
	}

	task := Task{
		ChatID:         chatID,
		Description:    description,
		CreatedAt:      time.Now(),
		Deadline:       time.Time{},
		Mark:           false,
		ReminderExists: false,
	}
	_, err := bs.db.InsertOne(context.TODO(), task)
	if err != nil {
		log.Printf("Failed to insert task: %s", err)
		bs.SendMessage(chatID, "Не удалось добавить задачу.")
		return
	}

	bs.SendMessage(chatID, "Задача добавлена!")
}

func (bs *BotService) EditTask(chatID int64, text string) {
	if text == "" {
		bs.SendMessage(chatID, "Пожалуйста, укажите описание старой и новой задачи.")
		return
	}
	parts := strings.SplitN(text, "|", 2)
	if len(parts) != 2 {
		bs.SendMessage(chatID, "Неверный формат команды. Используйте: /edit <старое описание задачи>|<новое описание задачи>")
		return
	}

	if text == "" {
		bs.SendMessage(chatID, "Введите текст задачи, которую хотите изменить")
		return
	}

	oldText := strings.TrimSpace(parts[0])
	newText := strings.TrimSpace(parts[1])

	if oldText == "" || newText == "" {
		bs.SendMessage(chatID, "Пожалуйста, укажите старое и новое описание задачи.")
		return
	}

	filter := bson.M{"chat_id": chatID, "description": oldText}
	update := bson.M{"$set": bson.M{"description": newText}}

	result, err := bs.db.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Printf("Failed to find task: %s", err)
		return
	}

	if result.ModifiedCount == 0 && text != "" {
		bs.SendMessage(chatID, "Задача с таким описанием не найдена, или не было изменений.")
		return
	}

	bs.SendMessage(chatID, "Задача успешно изменена!")
}

func (bs *BotService) SetDeadline(chatID int64, text string, client *asynq.Client) {
	if text == "" {
		bs.SendMessage(chatID, "Пожалуйста, укажите описание задачи и дедлайн.")
		return
	}
	parts := strings.SplitN(text, "|", 2)
	if len(parts) != 2 {
		bs.SendMessage(chatID, "Неверный формат команды. Используйте: /set_deadline <описание задачи> | <дата в формате YYYY-MM-DD HH:MM>")
		return
	}

	taskText := strings.TrimSpace(parts[0])
	deadlineStr := strings.TrimSpace(parts[1])

	deadlineTime, err := time.Parse("2006-01-02 15:04", deadlineStr)
	if err != nil {
		bs.SendMessage(chatID, "Неверный формат даты. Используйте формат: YYYY-MM-DD HH:MM")
		return
	}

	filter := bson.M{"chat_id": chatID, "description": taskText}
	update := bson.M{"$set": bson.M{"deadline": deadlineTime}}

	cursor, err := bs.db.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("Failed to list tasks: %s", err)
		bs.SendMessage(chatID, "Не удалось получить список задач.")
		return
	}
	defer cursor.Close(context.TODO())

	var tasks []Task
	if err := cursor.All(context.TODO(), &tasks); err != nil {
		log.Printf("Failed to decode tasks: %s", err)
		bs.SendMessage(chatID, "Не удалось получить список задач.")
		return
	}

	result, err := bs.db.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Printf("Failed to update task: %s", err)
		bs.SendMessage(chatID, "Не удалось установить дедлайн.")
		return
	}

	if result.ModifiedCount == 0 && text != "" {
		bs.SendMessage(chatID, "Задача не найдена или дедлайн не был изменен.")
		return
	}

	// payload, err := json.Marshal(taskToUpdate)
	// if err != nil {
	// log.Printf("Error mashalling Task to json: %s", err)
	// }

	// // Calculate the delay until the deadline
	// delay := time.Until(deadlineTime)

	// // Enqueue the task with the given parameters.
	// _, err = client.Enqueue("reminder:send", payload, asynq.ProcessAt(time.Now().Add(delay)))
	// if err != nil {
	// log.Fatalf("could not enqueue task: %v", err)
	// }

	bs.SendMessage(chatID, "Дедлайн установлен на "+deadlineTime.Format("2006-01-02 15:04")+"!")
}

func (bs *BotService) IsDone(chatID int64, text string) {
	if text == "" {
		bs.SendMessage(chatID, "Введите описание задачи, которую нужно отметить как выполненную.")
	}
	filter := bson.M{"chat_id": chatID, "description": text}

	_, err := bs.db.Find(context.TODO(), filter)
	if err != nil {
		bs.SendMessage(chatID, "Задача с таким описанием не найдена.")
		log.Printf("Failed to find task: %s", err)
		return
	}

	update := bson.M{"$set": bson.M{"mark": true, "reminder": false}}

	result, err := bs.db.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Printf("Failed to mark task: %s", err)
	}

	if result.ModifiedCount == 0 && text != "" {
		bs.SendMessage(chatID, "Задача с таким описанием не найдена, или не было изменений.")
		return
	}
	bs.SendMessage(chatID, "Задача выполнена!")
}

func (bs *BotService) StartReminder(intervalMinutes int, client *asynq.Client) {
	interval := time.Duration(intervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		bs.CheckDeadlines(client)
	}
}

func (bs *BotService) CheckDeadlines(client *asynq.Client) {
	yellow := color.New(color.FgYellow).SprintFunc()
	log.Println(yellow("Checking deadlines..."))
	filter := bson.M{}

	cursor, err := bs.db.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("Failed to retrieve tasks for deadline check: %v", err)
		return
	}
	defer cursor.Close(context.TODO())

	var tasks []Task
	if err := cursor.All(context.TODO(), &tasks); err != nil {
		log.Printf("Failed to decode tasks: %v", err)
		return
	}

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Failed to load location: %v", err)
		return
	}

	now := time.Now().In(location)

	for _, task := range tasks {
		timeLeft := time.Until(task.Deadline)
		timeUntilDead := task.Deadline.Sub(now)

		if !task.Deadline.IsZero() && task.ReminderExists {
			days := int(timeLeft.Hours()) / 24
			hours := int(timeLeft.Hours()) % 24
			minutes := int(timeLeft.Minutes()) % 60
			timeUntilDeadline := fmt.Sprintf(" (Осталось: %d дн. %d ч. %d мин.)", days, hours, minutes)

			if timeUntilDead > 0 {
				bs.SendMessage(task.ChatID, fmt.Sprintf("Напоминание: Скоро дедлайн по задаче \"%s\"! %s.", task.Description, timeUntilDeadline))
			}
		}
		if timeUntilDead <= 0 && !task.Deadline.IsZero() {
			newDeadline := now.Add(24 * time.Hour)
			filter := bson.M{"chat_id": task.ChatID, "description": task.Description}
			update := bson.M{"$set": bson.M{"deadline": newDeadline}}

			result, err := bs.db.UpdateOne(context.TODO(), filter, update)
			if err != nil {
				log.Printf("Failed to update task: %s", err)
				bs.SendMessage(task.ChatID, "Не удалось установить дедлайн.")
				return
			}

			if result.ModifiedCount == 0 {
				bs.SendMessage(task.ChatID, "Задача не найдена или дедлайн не был изменен.")
				return
			}

			bs.SendMessage(task.ChatID, fmt.Sprintf("Дедлайн по задаче '%s' истёк. Дедлайн перенесён на завтра", task.Description))
		}
	}

}

func (bs *BotService) SetReminder(chatID int64, text string, setReminder bool, client *asynq.Client) {
	if text == "" {
		bs.SendMessage(chatID, "Пожалуйста, укажите описание задачи.")
		return
	}
	filter := bson.M{"chat_id": chatID, "description": text}
	update := bson.M{"$set": bson.M{"reminder": setReminder}}

	result, err := bs.db.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Printf("Failed to remind task: %s", err)
		bs.SendMessage(chatID, "Не удалось установить/отменить напоминание.")
	}

	if result.ModifiedCount == 0 {
		bs.SendMessage(chatID, "Задача с таким описанием не найдена, или не было изменений.")
		return
	}

	if setReminder {
		bs.SendMessage(chatID, "Напоминание успешно установлено!")
	} else {
		bs.SendMessage(chatID, "Напоминание успешно отменено!")
	}
}

func (bs *BotService) ShowStats(chatID int64) {
	stats, err := bs.GetTaskStatistics(chatID)
	if err != nil {
		log.Printf("Failed to retrieve statistics: %v", err)
		bs.SendMessage(chatID, "Не удалось получить статистику.")
		return
	}

	message := fmt.Sprintf("Статистика по вашим задачам:\n"+
		"Выполнено вовремя: %d\n"+
		"Просрочено: %d\n"+
		"Средний срок установки дедлайна: %.2f дней\n",
		stats.CompletedOnTime, stats.Overdue, stats.AverageDeadlineDays)

	bs.SendMessage(chatID, message)
}

func (bs *BotService) GetTaskStatistics(chatID int64) (TaskStatistics, error) {
	var stats TaskStatistics

	filter := bson.M{"chat_id": chatID}

	cursor, err := bs.db.Find(context.TODO(), filter)
	if err != nil {
		return stats, err
	}
	defer cursor.Close(context.TODO())

	var tasks []Task
	if err := cursor.All(context.TODO(), &tasks); err != nil {
		return stats, err
	}

	var totalDeadlineDays float64
	var deadlineCount int

	for _, task := range tasks {
		if !task.Deadline.IsZero() {
			deadlineCount++
			timeUntilDead := task.Deadline.Sub(task.CreatedAt).Hours() / 24
			totalDeadlineDays += timeUntilDead
		}

		if task.Mark {
			if !task.Deadline.IsZero() && task.Deadline.Before(time.Now()) {
				stats.Overdue++
			} else {
				stats.CompletedOnTime++
			}
		}
	}

	if deadlineCount > 0 {
		stats.AverageDeadlineDays = totalDeadlineDays / float64(deadlineCount)
	}

	return stats, nil
}

func (bs *BotService) ChooseMethod(chatID int64, command string, text string, client *asynq.Client) {
	switch command {
	case "add":
		bs.AddTask(chatID, text, client)
	case "set_deadline":
		bs.SetDeadline(chatID, text, client)
	case "list":
		bs.ListTasks(chatID)
	case "delete":
		bs.DeleteTask(chatID, text)
	case "edit":
		bs.EditTask(chatID, text)
	case "is_done":
		bs.IsDone(chatID, text)
	case "set_reminder":
		bs.SetReminder(chatID, text, true, client)
	case "unset_reminder":
		bs.SetReminder(chatID, text, false, client)
	}
}
