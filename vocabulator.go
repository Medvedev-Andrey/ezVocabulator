package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	maxContentLength             = 4096
	StoreDictionaryRequestPrefix = "/slm"
)

type dictionaryResponse struct {
	entries []dictionaryEntry
}

type dictionaryEntry struct {
	item           string
	pronunciations []entryPronunciation
	lexemes        []entryLexeme
}

type entryPronunciation struct {
	regions        []string
	transcriptions []string
	audioUrl       string
}

type entryLexeme struct {
	lemma        string
	partOfSpeech string
	definitions  []dictionaryItemData
}

type dictionaryItemData struct {
	Definition string   `json:"definition"`
	Examples   []string `json:"examples"`
	Synonyms   []string `json:"synonyms"`
	Antonyms   []string `json:"antonyms"`
}

type trainingData struct {
	ItemData  dictionaryItemData `json:"data"`
	Item      string             `json:"item"`
	Iteration int                `json:"iteration"`
}

func processRequest(request *http.Request) ([]byte, error) {
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Printf("Failed processing HTTP request")
		return nil, err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed processing response body")
		return nil, err
	}

	return contents, nil
}
