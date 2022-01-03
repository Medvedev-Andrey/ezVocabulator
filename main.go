package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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

	log.Print("Setting up Telegram webhook")
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

	log.Print("Setting up Database")
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

		switch update.Message.Text {
		case "/history":
			handleHistoryRequest(update.Message)
		default:
			handleDictionaryRequest(update.Message)
		}
	}
}

func handleHistoryRequest(inMessage *tgbotapi.Message) {
	log.Printf("Handling history request from user with ID %d", inMessage.From.ID)

	userRequests, err := getUserRequests(db, inMessage.From.ID)
	if err != nil {
		handleErrorWithReply(inMessage, err)
		return
	} else if len(userRequests) == 0 {
		sendSimpleReply(inMessage, "Seems like you have not requested any dictionary info yet ... 😞")
		return
	}

	file, err := ioutil.TempFile("", fmt.Sprintf("%d_user_history.txt", inMessage.From.ID))
	if err != nil {
		handleErrorWithReply(inMessage, err)
		return
	}
	defer os.Remove(file.Name())

	for _, userRequest := range userRequests {
		_, err := file.WriteString(userRequest + "\n")
		if err != nil {
			handleErrorWithReply(inMessage, err)
			return
		}
	}

	if err != nil {
		handleErrorWithReply(inMessage, err)
	} else {
		msg := tgbotapi.NewDocumentUpload(inMessage.Chat.ID, file)
		msg.ReplyToMessageID = inMessage.MessageID

		_, err := bot.Send(msg)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleDictionaryRequest(inMessage *tgbotapi.Message) {
	log.Printf("Handling dictionary request '%s' from user with ID %d", inMessage.Text, inMessage.From.ID)

	lrResponse, err := getDefinitionFromLinguaRobot(inMessage.Text)
	if err != nil {
		handleErrorWithReply(inMessage, err)
	} else if len(lrResponse.Entries) == 0 {
		sendSimpleReply(inMessage, "Nothing has been found ... 😞")
	} else {
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
	}
}

func handleErrorWithReply(inMessage *tgbotapi.Message, err error) {
	log.Println(err)
	sendSimpleReply(inMessage, "Failed processing request ... 🤔")
}

func sendSimpleReply(inMessage *tgbotapi.Message, message string) {
	msg := tgbotapi.NewMessage(inMessage.Chat.ID, message)
	msg.ReplyToMessageID = inMessage.MessageID

	if _, err := bot.Send(msg); err != nil {
		log.Fatal(err)
	}
}
