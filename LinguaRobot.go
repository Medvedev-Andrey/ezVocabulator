package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	linguaRobotApiToken string
)

const (
	linguaRobotRequestFormat = "https://lingua-robot.p.rapidapi.com/language/v1/entries/en/%s"
	linguaRobotApiHost       = "lingua-robot.p.rapidapi.com"
	maxExamples              = 3
	maxSenses                = 5
)

type linguaRobotResponse struct {
	Entries []linguaRobotEntry `json:"entries"`
}

type linguaRobotEntry struct {
	Entry          string                      `json:"entry"`
	Pronunciations []linguaRobotPronunciations `json:"pronunciations"`
	Lexemes        []linguaRobotLexeme         `json:"lexemes"`
}

type linguaRobotPronunciations struct {
	Transcriptions []linguaRobotTranscription `json:"transcriptions"`
	Audio          linguaRobotAudio           `json:"audio"`
	Context        linguaRobotContext         `json:"context"`
}

type linguaRobotTranscription struct {
	Transcription string `json:"transcription"`
	Notation      string `json:"notation"`
}

type linguaRobotAudio struct {
	Url string `json:"url"`
}

type linguaRobotContext struct {
	Regions []string `json:"regions"`
}

type linguaRobotLexeme struct {
	Lemma        string             `json:"lemma"`
	PartOfSpeech string             `json:"partOfSpeech"`
	Senses       []linguaRobotSense `json:"senses"`
}

type linguaRobotSense struct {
	Definition string   `json:"definition"`
	Examples   []string `json:"usageExamples"`
	Synonyms   []string `json:"synonyms"`
	Antonyms   []string `json:"antonyms"`
}

func getDefinitionFromLinguaRobot(item string) (*linguaRobotResponse, error) {
	if linguaRobotApiToken == "" {
		linguaRobotApiToken = os.Getenv("LINGUA_ROBOT_API_TOKEN")
		if linguaRobotApiToken == "" {
			log.Fatalf("Environment variable for Lingua Robot is not set")
			return nil, fmt.Errorf("Empty Lingua Robot API token")
		}
	}

	item = strings.ToLower(item)
	requestUrl := fmt.Sprintf(linguaRobotRequestFormat, url.PathEscape(item))
	request, _ := http.NewRequest("GET", requestUrl, nil)
	request.Header.Add("x-rapidapi-host", linguaRobotApiHost)
	request.Header.Add("x-rapidapi-key", linguaRobotApiToken)

	contents, err := processRequest(request)
	if err != nil {
		fmt.Printf("Failed getting meanings from Lingua Robot for '%s'", item)
		return nil, err
	}

	var response linguaRobotResponse
	err = json.Unmarshal(contents, &response)
	if err != nil {
		fmt.Printf("Failed response deserialization from Lingua Robot for '%s'", item)
		return nil, err
	}

	return &response, nil
}

func convertLinguaRobotResponse(lrResponse *linguaRobotResponse) *dictionaryResponse {
	var response dictionaryResponse

	for _, lrEntry := range lrResponse.Entries {
		var entry dictionaryEntry

		entry.item = lrEntry.Entry

		for _, lrPronunciation := range lrEntry.Pronunciations {
			var pronunciation entryPronunciation

			pronunciation.audioUrl = lrPronunciation.Audio.Url
			pronunciation.regions = lrPronunciation.Context.Regions

			for _, lrTranscription := range lrPronunciation.Transcriptions {
				transcription := fmt.Sprintf("%s (%s)", lrTranscription.Transcription, lrTranscription.Notation)
				pronunciation.transcriptions = append(pronunciation.transcriptions, transcription)
			}

			entry.pronunciations = append(entry.pronunciations, pronunciation)
		}

		for _, lrLexeme := range lrEntry.Lexemes {
			var lexeme entryLexeme

			lexeme.lemma = lrLexeme.Lemma
			lexeme.partOfSpeech = lrLexeme.PartOfSpeech

			for _, lrSense := range lrLexeme.Senses {
				var itemData dictionaryItemData

				itemData.Definition = lrSense.Definition
				itemData.Antonyms = lrSense.Antonyms
				itemData.Synonyms = lrSense.Synonyms
				itemData.Examples = lrSense.Examples

				lexeme.definitions = append(lexeme.definitions, itemData)
			}

			entry.lexemes = append(entry.lexemes, lexeme)
		}

		response.entries = append(response.entries, entry)
	}

	return &response
}
