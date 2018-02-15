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
	"github.com/dzyubspirit/telegram-bot-api"
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

type Mention struct {
	Text string
	Type string         `json:"type"`
	URL  string         `json:"url"`  // optional
	User *tgbotapi.User `json:"user"` // optional
}

func InsertMentions(strs []string, mentions ...Mention) (string, []tgbotapi.MessageEntity) {
	if len(strs) != len(mentions)+1 {
		panic("len(strs) should be equal len(metions) + 1")
	}
	entities := make([]tgbotapi.MessageEntity, len(mentions))
	var buf bytes.Buffer
	for i, mention := range mentions {
		entities[i] = tgbotapi.MessageEntity{
			Type:   mention.Type,
			URL:    mention.URL,
			Offset: buf.Len(),
			Length: len(mention.Text),
			User:   mention.User,
		}
		buf.WriteString(strs[i])
		buf.WriteString(mention.Text)
	}
	buf.WriteString(strs[len(strs)-1])
	return buf.String(), entities
}

func NewMention(text string, mention tgbotapi.MessageEntity) Mention {
	return Mention{
		Text: text,
		URL:  mention.URL,
		Type: mention.Type,
		User: mention.User,
	}
}

type User struct {
	Mention Mention
	Type    string
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

	var mention *Mention
	for _, ent := range *update.Message.Entities {
		if ent.Type == "mention" || ent.Type == "text_mention" {
			text := string([]rune(update.Message.Text)[ent.Offset:ent.Offset+ent.Length])
			mention = new(Mention)
			*mention = NewMention(text, ent)
			break
		}
	}
	if mention == nil {
		return "Упомяни человека в команде боту", nil
	}

	users[mention.Text] = User{Mention: *mention, Type: typ}
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		var buffer bytes.Buffer
		err := gob.NewEncoder(&buffer).Encode(users[mention.Text])
		if err != nil {
			return fmt.Errorf("error encoding user: %v", err)
		}

		return b.Put([]byte(mention.Text), buffer.Bytes())
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s - %s", mention.Text, typ), nil
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
		var entities []tgbotapi.MessageEntity
		switch complience[pair.From.Type][pair.To.Type] {
		case dual:
			msg, entities = InsertMentions([]string{"", " влюблен(а) 😍😍😍😍😍😍😍 в ", ""}, pair.From.Mention, pair.To.Mention)
		case activator:
			msg, entities = InsertMentions([]string{"", " влюблен(а) 😍😍😍😍 в ", ", но боиться признаться в этом 🙈"}, pair.From.Mention, pair.To.Mention)
		case halfDual:
			msg, entities = InsertMentions([]string{"", " немного влюблен(а) 😍😍😍 в ", ""}, pair.From.Mention, pair.To.Mention)
		}
		mc := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
		mc.Entities = entities
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
