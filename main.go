package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
)

var (
	bot *tgbotapi.BotAPI
	db  *sql.DB
)

const (
	appURL = "https://ezvocabulator.herokuapp.com/"
)

func initTelegram(botToken string) (*tgbotapi.BotAPI, error) {
	log.Print("Setting up Telegram API connection")

	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	url := appURL + bot.Token
	_, err = bot.SetWebhook(tgbotapi.NewWebhook(url))
	if err != nil {
		return nil, err
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}

	return bot, nil
}

func main() {
	botToken := os.Getenv("TELEGRAM_API_TOKEN")
	if botToken == "" {
		log.Fatalf("Environment variable for Telegram API is not set")
	}

	port := os.Getenv("PORT")
	if botToken == "" {
		log.Fatalf("Environment variable for Port is not set")
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatalf("Environment variable for Database is not set")
	}

	var err error
	db, err = sql.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	err = ensureDictionaryRequestDBExists(db)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := initTelegram(botToken)
	if err != nil {
		log.Fatal(err)
	}

	updates := bot.ListenForWebhook("/" + bot.Token)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	go http.ListenAndServe(addr, nil)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Text == "\\history" {
			handleHistoryRequest(update.Message)
		} else {
			handleDictionaryRequest(update.Message)
		}
	}
}

func handleHistoryRequest(inMessage *tgbotapi.Message) {
	log.Printf("Handling history request from user with ID %d", inMessage.From.ID)

	userRequests, err := getUserRequests(db, inMessage.From.ID)
	if err == nil && len(userRequests) > 0 {
		msg := tgbotapi.NewMessage(inMessage.Chat.ID, "")
		msg.ReplyToMessageID = inMessage.MessageID
		msg.Text = strings.Join(userRequests, ", ")

		_, err := bot.Send(msg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var message string
		if err != nil {
			log.Println(err)
			message = "Failed processing request ... 🤔"
		} else {
			message = "Seems like you have not requested any dictionary info yet ... 😞"
		}

		msg := tgbotapi.NewMessage(inMessage.Chat.ID, message)
		msg.ReplyToMessageID = inMessage.MessageID

		if _, err := bot.Send(msg); err != nil {
			log.Fatal(err)
		}
	}
}

func handleDictionaryRequest(inMessage *tgbotapi.Message) {
	log.Printf("Handling dictionary request '%s' from user with ID %d", inMessage.Text, inMessage.From.ID)

	lrResponse, err := getDefinitionFromLinguaRobot(inMessage.Text)
	if err == nil && len(lrResponse.Entries) > 0 {
		response := convertLinguaRobotResponse(lrResponse)
		contents := formatUserResponse(response)

		storeDictionaryRequest(db, inMessage.From.ID, inMessage.Text)

		messageIDToReply := inMessage.MessageID
		for _, content := range contents {
			msg := tgbotapi.NewMessage(inMessage.Chat.ID, "")
			msg.ReplyToMessageID = messageIDToReply
			msg.ParseMode = "HTML"
			msg.Text = content

			sentMsg, err := bot.Send(msg)
			if err != nil {
				log.Fatal(err)
			}

			messageIDToReply = sentMsg.MessageID
		}
	} else {
		var message string
		if err != nil {
			log.Println(err)
			message = "Failed processing request ... 🤔"
		} else {
			message = "Nothing has been found ... 😞"
		}

		msg := tgbotapi.NewMessage(inMessage.Chat.ID, message)
		msg.ReplyToMessageID = inMessage.MessageID

		if _, err := bot.Send(msg); err != nil {
			log.Fatal(err)
		}
	}
}
