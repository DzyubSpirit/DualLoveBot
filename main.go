package main

import (
	"log"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
	"strings"
	"fmt"
	"math/rand"
	"flag"
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
	Type string
}

var users = map[string]User{}

func addUser(bot *tgbotapi.BotAPI, update tgbotapi.Update) string {
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
		return "Укажите социотип, плез"
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
		return "Упомяни человека в команде боту"
	}

	users[nick] = User{nick,typ}
	return fmt.Sprintf("%s - %s", nick, typ)
}

func handleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	switch update.Message.Command() {
	case "add":
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		msg := addUser(bot, update)
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

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		handleCommand(bot, update)
		/*
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		bot.Send(msg)
		*/
	}
}
