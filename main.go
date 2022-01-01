package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	appURL = "https://ezvocabulator.herokuapp.com/"
)

func initTelegram(botToken string) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	url := appURL + bot.Token
	_, err = bot.SetWebhook(tgbotapi.NewWebhook(url))
	if err != nil {
		log.Fatal(err)
	}

	return bot
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

	bot := initTelegram(botToken)

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}

	updates := bot.ListenForWebhook("/" + bot.Token)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	go http.ListenAndServe(addr, nil)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		lrResponse, err := getDefinitionFromLinguaRobot(update.Message.Text)
		if err == nil {
			response := convertLinguaRobotResponse(lrResponse)
			contents := formatUserResponse(response)

			messageIDToReply := update.Message.MessageID
			for _, content := range contents {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
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
			log.Println(err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Failed processing request ... 🤔")
			msg.ReplyToMessageID = update.Message.MessageID

			if _, err := bot.Send(msg); err != nil {
				log.Fatal(err)
			}
		}
	}
}
