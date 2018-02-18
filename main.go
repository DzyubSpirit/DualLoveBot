package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/boltdb/bolt"
	"gopkg.in/telegram-bot-api.v4"
	"strconv"
)

const botName = "@duallovebot"

const (
	dual      = 1
	activator = 2
	halfDual  = 3
)

var (
	socioTypes = []string{"Дон Кихот", "Дюма", "Гюго", "Робеспьер", "Гамлет", "Максим", "Жуков", "Есенин", "Наполеон",
		"Бальзак", "Джек", "Драйзер", "Штирлиц", "Достоевский", "Гексли", "Габен",
	}
	complience = map[string]map[string]int{
		"Дон Кихот":   {"Дюма": dual, "Гюго": activator, "Габен": halfDual},
		"Дюма":        {"Дон Кихот": dual, "Робеспьер": activator, "Гексли": halfDual},
		"Гюго":        {"Робеспьер": dual, "Дон Кихот": activator, "Максим": halfDual},
		"Робеспьер":   {"Гюго": dual, "Дюма": activator, "Гамлет": halfDual},
		"Гамлет":      {"Максим": dual, "Жуков": activator, "Робеспьер": halfDual},
		"Максим":      {"Гамлет": dual, "Есенин": activator, "Гюго": halfDual},
		"Жуков":       {"Есенин": dual, "Гамлет": activator, "Бальзак": halfDual},
		"Есенин":      {"Жуков": dual, "Максим": activator, "Наполеон": halfDual},
		"Наполеон":    {"Бальзак": dual, "Джек": activator, "Есенин": halfDual},
		"Бальзак":     {"Наполеон": dual, "Драйзер": activator, "Жуков": halfDual},
		"Джек":        {"Драйзер": dual, "Наполеон": activator, "Достоевский": halfDual},
		"Драйзер":     {"Джек": dual, "Бальзак": activator, "Штирлиц": halfDual},
		"Штирлиц":     {"Достоевский": dual, "Гексли": activator, "Драйзер": halfDual},
		"Достоевский": {"Штирлиц": dual, "Габен": activator, "Джек": halfDual},
		"Гексли":      {"Габен": dual, "Штирлиц": activator, "Дюма": halfDual},
		"Габен":       {"Гексли": dual, "Достоевский": activator, "Дон Кихот": halfDual},
	}
)

var botKeyVar string

type User struct {
	DisplayName string
	Type        string
}

type UserWithID struct {
	User
	ID int
}

type Mentioner interface {
	Mention() string
	GetType() string
}

func (u User) Mention() string {
	return fmt.Sprintf("[%s]", u.DisplayName)
}

func (u User) GetType() string {
	return u.Type
}

func (u UserWithID) Mention() string {
	return fmt.Sprintf("[%s](tg://user?id=%v)", u.DisplayName, u.ID)
}

type UserIDType = int

var users = map[UserIDType]Mentioner{}

func addUser(db *bolt.DB, update tgbotapi.Update) (string, error) {
	msg := update.Message.Text
	lMsg := strings.ToLower(msg)

	var typ string
	for _, t := range socioTypes {
		if strings.Contains(lMsg, strings.ToLower(t)) {
			typ = t
			break
		}
	}
	if typ == "" {
		return "Укажите социотип, плез", nil
	}

	var displayName string
	var userID int
	for _, ent := range *update.Message.Entities {
		if ent.Type == "mention" || ent.Type == "text_mention" {
			text := string([]rune(update.Message.Text)[ent.Offset:ent.Offset+ent.Length])
			displayName = text
			if ent.User != nil {
				userID = ent.User.ID
			}
			break
		}
	}
	if displayName == "" {
		return "Упомяни человека в команде боту", nil
	}

	u := User{DisplayName: displayName, Type: typ}
	if userID != 0 {
		users[userID] = UserWithID{User: u, ID: userID}
	} else {
		users[userID] = u
	}
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		var buffer bytes.Buffer
		err := gob.NewEncoder(&buffer).Encode(users[userID])
		if err != nil {
			return fmt.Errorf("error encoding user: %v", err)
		}

		return b.Put([]byte(strconv.Itoa(userID)), buffer.Bytes())
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s - %s", displayName, typ), nil
}

type Pair struct {
	From Mentioner
	To   Mentioner
}

var lastPair *Pair

func handleCommand(db *bolt.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	switch update.Message.Command() {
	case "add":
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		msg, err := addUser(db, update)
		if err != nil {
			log.Printf("ERROR: command \"add\": %v", err)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Поздравяю! Ты сломал бота! Ну кто тебя просил то... Напиши ॅॅ@Vladka_Marmelaka об этом"))
			return
		}
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
	case "joke":
		pairs := make([]Pair, 0, len(users)*len(users))
		for _, u1 := range users {
			for _, u2 := range users {
				if complience[u1.GetType()][u2.GetType()] > 0 {
					pairs = append(pairs, Pair{u1, u2})
				}
			}
		}

		if len(pairs) < 1 {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Никто никого не любит :("))
			return
		}

		pair := pairs[rand.Intn(len(pairs))]
		lastPair = &pair

		from := pair.From.Mention()
		to := pair.To.Mention()

		log.Printf("from: %q, to: %q", from, to)
		var msg string
		switch complience[pair.From.GetType()][pair.To.GetType()] {
		case dual:
			msg = fmt.Sprintf("%s влюблен(а) 😍😍😍😍😍😍😍 в %s", from, to)
		case activator:
			msg = fmt.Sprintf("%s влюблен(а) 😍😍😍😍 в %s, но боиться признаться в этом 🙈", from, to)
		case halfDual:
			msg = fmt.Sprintf("%s немного влюблен(а) 😍😍😍 в %s", from, to)
		}
		mc := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
		mc.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(mc)
	case "not_a_joke":
		if lastPair == nil {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "There was no jokes"))
			return
		}

		pair := *lastPair
		from := pair.From.Mention()
		to := pair.To.Mention()
		msg := fmt.Sprintf("❤️❤️❤️ %s ❤️❤️❤️\n тебя приглашает на свидание %s 😱😱😱", from, to)
		mc := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
		mc.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(mc)
	}

}

func main() {
	flag.StringVar(&botKeyVar, "bot_key", "", "bot API key")
	flag.Parse()
	if botKeyVar == "" {
		log.Fatalf("should bot_key parameter")
	}

	bot, err := tgbotapi.NewBotAPI(botKeyVar)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatalf("error openning database my.db: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("error creating bucket: %v", err)
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			key, _ := strconv.Atoi(string(k))
			var val Mentioner
			var val1 User
			var val2 UserWithID

			err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(&val1)
			val = val1
			if err != nil {
				err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(&val2)
				val = val2
			}
			if err != nil {
				return fmt.Errorf("error decoding val %s, err: %v", v, err)
			}

			users[key] = val
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	updates, err := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		handleCommand(db, bot, update)
		/*
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		*/
	}
}
