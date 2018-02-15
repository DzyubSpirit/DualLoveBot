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
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const botName = "@duallovebot"

const (
	dual     = 1
	halfDual = 2
)

var (
	socioTypes = []string{"Дон Кихот", "Дюма", "Гюго", "Робеспьер", "Гамлет", "Максим", "Жуков", "Есенин", "Наполеон",
		"Бальзак", "Джек", "Драйзер", "Штирлиц", "Достоевский", "Гексли", "Габен",
	}
	complience = map[string]map[string]int{
		"Дон Кихот":   {"Дюма": dual, "Габен": halfDual},
		"Дюма":        {"Дон Кихот": dual, "Гексли": halfDual},
		"Гюго":        {"Робеспьер": dual, "Максим": halfDual},
		"Робеспьер":   {"Гюго": dual, "Гамлет": halfDual},
		"Гамлет":      {"Максим": dual, "Робеспьер": halfDual},
		"Максим":      {"Гамлет": dual, "Гюго": halfDual},
		"Жуков":       {"Есенин": dual, "Бальзак": halfDual},
		"Есенин":      {"Жуков": dual, "Наполеон": halfDual},
		"Наполеон":    {"Бальзак": dual, "Есенин": halfDual},
		"Бальзак":     {"Наполеон": dual, "Жуков": halfDual},
		"Джек":        {"Драйзер": dual, "Достоевский": halfDual},
		"Драйзер":     {"Джек": dual, "Штирлиц": halfDual},
		"Штирлиц":     {"Достоевский": dual, "Драйзер": halfDual},
		"Достоевский": {"Штирлиц": dual, "Джек": halfDual},
		"Гексли":      {"Габен": dual, "Дюма": halfDual},
		"Габен":       {"Гексли": dual, "Дон Кихот": halfDual},
	}
)

var botKeyVar string

type User struct {
	Nickname string
	Type     string
}

var users = map[string]User{}

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

	var nick string
	parts := strings.Split(msg, " ")
	for _, p := range parts {
		if p[0] == '@' && strings.ToLower(p) != botName {
			nick = strings.ToLower(p)
			break
		}
	}
	if nick == "" {
		return "Упомяни человека в команде боту", nil
	}

	users[nick] = User{nick, typ}
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		var buffer bytes.Buffer
		err := gob.NewEncoder(&buffer).Encode(users[nick])
		if err != nil {
			return fmt.Errorf("error encoding user: %v", err)
		}

		return b.Put([]byte(nick), buffer.Bytes())
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s - %s", nick, typ), nil
}

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
		type Pair struct {
			From User
			To   User
		}

		pairs := make([]Pair, 0, len(users)*len(users))
		for _, u1 := range users {
			for _, u2 := range users {
				if complience[u1.Type][u2.Type] > 0 {
					pairs = append(pairs, Pair{u1, u2})
				}
			}
		}

		if len(pairs) < 1 {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Никто никого не любит :("))
			return
		}

		pair := pairs[rand.Intn(len(pairs))]
		var msg string
		switch complience[pair.From.Type][pair.To.Type] {
		case dual:
			msg = fmt.Sprintf("%s влюблен(а) 😍😍😍😍😍😍😍 в %s", pair.From.Nickname, pair.To.Nickname)
		case halfDual:
			msg = fmt.Sprintf("%s немного влюблен(а) 😍😍😍 в %s", pair.From.Nickname, pair.To.Nickname)
		}
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
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
			key := string(k)
			var val User

			err = gob.NewDecoder(bytes.NewBuffer(v)).Decode(&val)
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