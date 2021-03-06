package main

import (
	"bytes"
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
	queryCacheHoursLifeSpan = 1
	appURL                  = "https://ezvocabulator.herokuapp.com/"
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

	err = ensureTrainingTableExists(db)
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

		if strings.HasPrefix(update.Message.Text, StoreTrainingDataPrefix) {
			handleStoreTrainingDataQuery(update.Message)
			continue
		}

		switch update.Message.Text {
		case "/history":
			handleUserTrainingDataRequest(update.Message)
		default:
			handleDictionaryRequest(update.Message)
		}
	}
}

func handleStoreTrainingDataQuery(inMessage *tgbotapi.Message) {
	trainingData, err := getTrainingData(inMessage.Text)
	if err != nil {
		sendSimpleReply(inMessage, "Cannot find corresponding dictionary request ... ????\nThe request cache may be outdated. Try requesting it again ... I'll definetely find the word and store definitions for later! ????")
		return
	}

	err = storeTrainingData(db, inMessage.From.ID, trainingData)
	if err != nil {
		handleErrorWithReply(inMessage, err)
	} else {
		sendSimpleReply(inMessage, fmt.Sprintf("Stored '%s' ???", trainingData.Item))
		delete(queryToTrainingData, inMessage.Text)
	}
}

func handleUserTrainingDataRequest(inMessage *tgbotapi.Message) {
	userDataCount, err := countUserTrainingData(db, inMessage.From.ID)
	if err != nil {
		handleErrorWithReply(inMessage, err)
		return
	}

	userTrainingData, err := getUserDataToTrain(db, inMessage.From.ID, userDataCount)
	if err != nil {
		handleErrorWithReply(inMessage, err)
		return
	} else if len(userTrainingData) == 0 {
		sendSimpleReply(inMessage, "Seems like you have no training data yet ... ????")
		return
	}

	var buf bytes.Buffer
	for i, trainingItem := range userTrainingData {
		buf.WriteString(fmt.Sprintf("[%d] %s: %s\n", i, trainingItem.Item, trainingItem.ItemData.Definition))
	}

	file := tgbotapi.FileBytes{
		Name:  "training_set.txt",
		Bytes: buf.Bytes(),
	}

	msg := tgbotapi.NewDocumentUpload(inMessage.Chat.ID, file)
	msg.ReplyToMessageID = inMessage.MessageID

	_, err = bot.Send(msg)
	if err != nil {
		log.Fatal(err)
	}
}

func handleDictionaryRequest(inMessage *tgbotapi.Message) {
	mWResponse, err := getDefinitionFromMWDictionary(inMessage.Text)
	if err != nil {
		handleErrorWithReply(inMessage, err)
		return
	} else if len(mWResponse.Entries) == 0 {
		sendSimpleReply(inMessage, "Nothing has been found ... ????")
		return
	}

	responseContent := convertMWDictionaryResponse(mWResponse)

	messageIDToReply := inMessage.MessageID
	for _, responseContentPart := range splitResponseContents(responseContent.content, maxContentLength, '\n') {
		msg := tgbotapi.NewMessage(inMessage.Chat.ID, "")
		msg.ReplyToMessageID = messageIDToReply
		msg.ParseMode = "HTML"
		msg.Text = responseContentPart

		sentMsg, err := bot.Send(msg)
		if err != nil {
			log.Fatal(err)
		}

		cacheTrainingDataSet(responseContent.storeQueries)
		messageIDToReply = sentMsg.MessageID
	}
}

func handleErrorWithReply(inMessage *tgbotapi.Message, err error) {
	log.Println(err)
	sendSimpleReply(inMessage, "Failed processing request ... ????")
}

func sendSimpleReply(inMessage *tgbotapi.Message, message string) {
	msg := tgbotapi.NewMessage(inMessage.Chat.ID, message)
	msg.ReplyToMessageID = inMessage.MessageID

	if _, err := bot.Send(msg); err != nil {
		log.Fatal(err)
	}
}
