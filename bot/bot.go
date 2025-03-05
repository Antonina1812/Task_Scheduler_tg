package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

//TODO: если срок задачи истёк, перенести дедлайн на день и отправить сообщение пользователю
//TODO: подключить редис
//TODO: подключить очередь задач
//TODO: добавить нейронку или альтернативу

//TODO: разбить проект на несколько серверов
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

type BotService struct {
	api *tgbotapi.BotAPI
	db  *mongo.Collection
}

func NewBotService(api *tgbotapi.BotAPI, db *mongo.Collection) *BotService {
	return &BotService{
		api: api,
		db:  db,
	}
}

func (bs *BotService) HandleCommand(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	command := message.Command()
	text := message.CommandArguments()

	switch command {
	case "start":
		bs.SendMessage(chatID, "Привет! Я бот, который поможет тебе управлять задачами. Используй /help для просмотра доступных команд.")
	case "help":
		bs.SendMessage(chatID, "Доступные команды:\n/add <описание задачи> - добавить задачу\n/list - список задач\n/delete <дата в формате YYYY-MM-DD HH:MM> - удалить задачу\n/is_done <текст задачи> - отметить задачу, как выполненную\n/edit <старое описание задачи>|<новое описание задачи>\n/set_deadline <номер задачи> <дата в формате YYYY-MM-DD HH:MM>\n/set_reminder <описание задачи> - установить напоминание\n/unset_reminder <описание задачи> - отменить напоминание\n/help - помощь")
	case "add":
		bs.AddTask(chatID, text)
	case "set_deadline":
		bs.SetDeadline(chatID, text)
	case "list":
		bs.ListTasks(chatID)
	case "delete":
		bs.DeleteTask(chatID, text)
	case "edit":
		bs.EditTask(chatID, text)
	case "is_done":
		bs.IsDone(chatID, text)
	case "set_reminder":
		bs.SetReminder(chatID, text, true)
	case "unset_reminder":
		bs.SetReminder(chatID, text, false)
	default:
		bs.SendMessage(chatID, "Неизвестная команда. Используйте /help для просмотра доступных команд.")
	}
}

func (bs *BotService) SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bs.api.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %s", err)
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

	if result.DeletedCount == 0 {
		bs.SendMessage(chatID, "Задача не найдена.")
		return
	}

	bs.SendMessage(chatID, "Задача удалена!")
}

func (bs *BotService) AddTask(chatID int64, description string) {
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

	if result.ModifiedCount == 0 {
		bs.SendMessage(chatID, "Задача с таким описанием не найдена, или не было изменений.")
		return
	}

	bs.SendMessage(chatID, "Задача успешно изменена!")
}

func (bs *BotService) SetDeadline(chatID int64, text string) {
	parts := strings.SplitN(text, "|", 2)
	if len(parts) != 2 {
		bs.SendMessage(chatID, "Неверный формат команды. Используйте: /set_deadline <номер задачи> <дата в формате YYYY-MM-DD HH:MM>")
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

	if result.ModifiedCount == 0 {
		bs.SendMessage(chatID, "Задача не найдена или дедлайн не был изменен.")
		return
	}
	bs.SendMessage(chatID, "Дедлайн установлен на "+deadlineTime.Format("2006-01-02 15:04")+"!")
}

func (bs *BotService) IsDone(chatID int64, text string) {
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

	if result.ModifiedCount == 0 {
		bs.SendMessage(chatID, "Задача с таким описанием не найдена, или не было изменений.")
		return
	}
	bs.SendMessage(chatID, "Задача выполнена!")
}

func (bs *BotService) StartReminder(intervalMinutes int) {
	interval := time.Duration(intervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		bs.CheckDeadlines()
	}
}

func (bs *BotService) CheckDeadlines() {
	log.Println("Checking deadlines...")
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

	now := time.Now()

	for _, task := range tasks {
		timeLeft := time.Until(task.Deadline)

		if !task.Deadline.IsZero() && task.ReminderExists {
			days := int(timeLeft.Hours()) / 24
			hours := int(timeLeft.Hours()) % 24
			minutes := int(timeLeft.Minutes()) % 60
			timeUntilDeadline := fmt.Sprintf(" (Осталось: %d дн. %d ч. %d мин.)", days, hours, minutes)
			timeUntilDead := task.Deadline.Sub(now)

			if timeUntilDead > 0 {
				bs.SendMessage(task.ChatID, fmt.Sprintf("Напоминание: Скоро дедлайн по задаче \"%s\"! %s.", task.Description, timeUntilDeadline))
			}
		}
	}
}

func (bs *BotService) SetReminder(chatID int64, text string, setReminder bool) {
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
