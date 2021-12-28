package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	apiToken string = ""
)

const (
	xfApiHost              = "xf-english-dictionary1.p.rapidapi.com"
	xfEnglishDictionaryUrl = "https://" + xfApiHost + "/v1/dictionary"
)

type xfPronunciationAudio struct {
	Link  string `json:"link"`
	Label string `json:"label"`
}

type xfPronunciationTextual struct {
	Pronunciation string `json:"pronunciation"`
}

type xfPronunciationEntry struct {
	Entry   string                   `json:"entry"`
	Audio   []xfPronunciationAudio   `json:"audioFiles"`
	Textual []xfPronunciationTextual `json:"textual"`
}

type xfPronunciation struct {
	SectionID string                 `json:"sectionID"`
	Entries   []xfPronunciationEntry `json:"entries"`
}

type xfPhraseDefinition struct {
	Definition string   `json:"definition"`
	Examples   []string `json:"examples"`
}

type xfDefinition struct {
	Definition string   `json:"definition"`
	Examples   []string `json:"examples"`
	Synonyms   []string `json:"synonyms"`
	Antonyms   []string `json:"antonyms"`
}

type xfPhrase struct {
	Phrase       string               `json:"phrase"`
	PartOfSpeech string               `json:"partOfSpeech"`
	Definitions  []xfPhraseDefinition `json:"definitions"`
}

type xfInflectionalForm struct {
	Type    string   `json:"type"`
	Comment string   `json:"comment"`
	Forms   []string `json:"forms"`
}

type xfItem struct {
	Word                   string               `json:"word"`
	PartOfSpeech           string               `json:"partOfSpeech"`
	Comment                string               `json:"comment"`
	Definitions            []xfDefinition       `json:"definitions"`
	Synonyms               []string             `json:"synonyms"`
	Antonyms               []string             `json:"antonyms"`
	InflectionalForm       []xfInflectionalForm `json:"inflectionalForms"`
	PronunciationSectionID string               `json:"pronunciationSectionID"`
	Phrases                []xfPhrase           `json:"phrases"`
}

type xfDictionaryResponse struct {
	Target         string            `json:"target"`
	Items          []xfItem          `json:"items"`
	Pronunciations []xfPronunciation `json:"pronunciations"`
}

type xfDictionaryRequest struct {
	Text string `json:"selection"`
}

func getFromXfEnglishDictionary(word string) (*xfDictionaryResponse, error) {
	if apiToken == "" {
		apiToken = os.Getenv("XF_DICTIONARY_API_TOKEN")
		if apiToken == "" {
			log.Fatalf("Environment variable for XF Dictionary is not set")
			return nil, fmt.Errorf("Empty XF Dictionary API token")
		}
	}

	payload, _ := json.Marshal(xfDictionaryRequest{Text: word})
	request, _ := http.NewRequest("POST", xfEnglishDictionaryUrl, bytes.NewReader(payload))
	request.Header.Add("content-type", "application/json")
	request.Header.Add("x-rapidapi-host", xfApiHost)
	request.Header.Add("x-rapidapi-key", apiToken)

	contents, err := processRequest(request)
	if err != nil {
		fmt.Printf("Failed getting meanings for '%s'", word)
		return nil, err
	}

	var response xfDictionaryResponse
	err = json.Unmarshal(contents, &response)
	if err != nil {
		fmt.Printf("Failed XF Dictionary response meanings deserialization for '%s'", word)
		return nil, err
	}

	return &response, nil
}

func formatXfResponse(response *xfDictionaryResponse) (string, error) {
	var sb strings.Builder

	if len(response.Items) == 0 {
		sb.WriteString("Nothing has been found ... 😞")
		return sb.String(), nil
	}

	sb.WriteString("<b>Found entries:</b>\n\n")

	for _, item := range response.Items {
		sb.WriteString(fmt.Sprintf("▫️%s", item.Word))

		if item.PartOfSpeech != "" {
			sb.WriteString(fmt.Sprintf(" <i>(%s)</i>", item.PartOfSpeech))
		}

		sb.WriteRune('\n')

		for _, pronunciationEntry := range getXfPronunciations(response, &item) {
			for _, textualPronunciation := range pronunciationEntry.Textual {
				sb.WriteString(fmt.Sprintf("%s\n", textualPronunciation.Pronunciation))
			}
		}
	}

	return sb.String(), nil
}

func getXfPronunciations(response *xfDictionaryResponse, xfItem *xfItem) []*xfPronunciationEntry {
	var pronunciations []*xfPronunciationEntry

	if len(response.Pronunciations) == 0 {
		return pronunciations
	}

	if xfItem.PronunciationSectionID != "" {
		for _, prSection := range response.Pronunciations {
			if prSection.SectionID == xfItem.PronunciationSectionID {
				for _, pronunciationSectionEntry := range prSection.Entries {
					pronunciations = append(pronunciations, &pronunciationSectionEntry)
				}
			}
		}
	}

	//	If no pronunciations are found
	//	Append each pronunciation with same entry text
	if len(pronunciations) == 0 {
		for _, prSection := range response.Pronunciations {
			if prSection.SectionID == "" {
				for _, prSectionEntry := range prSection.Entries {
					if prSectionEntry.Entry == xfItem.Word {
						pronunciations = append(pronunciations, &prSectionEntry)
					}
				}
			}
		}
	}

	return pronunciations
}
