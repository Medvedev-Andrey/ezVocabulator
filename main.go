package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
)

var (
	bot *tgbotapi.BotAPI
	db  *sql.DB

	queryToTrainingData map[string]trainingData
	lastInputTimestamp  time.Time
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

		clearQueryCacheIfNeeded()
		lastInputTimestamp = time.Now()

		if strings.HasPrefix(update.Message.Text, StoreDictionaryRequestPrefix) {
			handleStoreDictionaryQuery(update.Message)
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

func clearQueryCacheIfNeeded() {
	if time.Since(lastInputTimestamp).Hours() > queryCacheHoursLifeSpan {
		for k := range queryToTrainingData {
			delete(queryToTrainingData, k)
		}
	}
}

func handleStoreDictionaryQuery(inMessage *tgbotapi.Message) {
	trainingData, ok := queryToTrainingData[inMessage.Text]
	if !ok {
		sendSimpleReply(inMessage, "Cannot find corresponding dictionary request ... 😞")
		return
	}

	err := storeTrainingData(db, inMessage.From.ID, &trainingData)
	if err != nil {
		handleErrorWithReply(inMessage, err)
	} else {
		sendSimpleReply(inMessage, "Stored '%s' definition ✅")
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
		sendSimpleReply(inMessage, "Seems like you have no training data yet ... 😞")
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
	lrResponse, err := getDefinitionFromLinguaRobot(inMessage.Text)
	if err != nil {
		handleErrorWithReply(inMessage, err)
		return
	} else if len(lrResponse.Entries) == 0 {
		sendSimpleReply(inMessage, "Nothing has been found ... 😞")
		return
	}

	response := convertLinguaRobotResponse(lrResponse)
	responseContents := formatUserResponse(response)

	messageIDToReply := inMessage.MessageID
	for _, responseContent := range responseContents {
		msg := tgbotapi.NewMessage(inMessage.Chat.ID, "")
		msg.ReplyToMessageID = messageIDToReply
		msg.ParseMode = "HTML"
		msg.Text = responseContent.content

		sentMsg, err := bot.Send(msg)
		if err != nil {
			log.Fatal(err)
		}

		for query, trainingData := range responseContent.storeQueries {
			queryToTrainingData[query] = trainingData
		}

		messageIDToReply = sentMsg.MessageID
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
