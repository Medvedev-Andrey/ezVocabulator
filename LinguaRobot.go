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

const (
	linguaRobotRequestFormat = "https://lingua-robot.p.rapidapi.com/language/v1/entries/en/%s"
	maxExamples              = 3
	maxSenses                = 5
)

type linguaRobotResponse struct {
	Entries []linguaRobotEntry `json:"entries"`
}

type linguaRobotEntry struct {
	Entry             string                      `json:"entry"`
	Pronunctionations []linguaRobotPronunciations `json:"pronunciations"`
	Lexemes           []linguaRobotLexeme         `json:"lexemes"`
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
	if apiToken == "" {
		apiToken = os.Getenv("LINGUA_ROBOT_API_TOKEN")
		if apiToken == "" {
			log.Fatalf("Environment variable for Lingua Robot is not set")
			return nil, fmt.Errorf("Empty Lingua Robot API token")
		}
	}

	item = strings.ToLower(item)
	requestUrl := fmt.Sprintf(linguaRobotRequestFormat, url.PathEscape(item))
	request, _ := http.NewRequest("GET", requestUrl, nil)
	request.Header.Add("x-rapidapi-host", "lingua-robot.p.rapidapi.com")
	request.Header.Add("x-rapidapi-key", apiToken)

	contents, err := processRequest(request)
	if err != nil {
		fmt.Printf("Failed getting meanings from Lingua Robot for '%s'", item)
		return nil, err
	}

	var response linguaRobotResponse
	err = json.Unmarshal(contents, &response)
	if err != nil {
		fmt.Printf("Failed XF Dictionary response meanings deserialization from Lingua Robot for '%s'", item)
		return nil, err
	}

	return &response, nil
}

func formatLinguaRobotResponse(response *linguaRobotResponse) (string, error) {
	var sb strings.Builder

	if len(response.Entries) == 0 {
		sb.WriteString("Nothing has been found ... 😞")
		return "", nil
	}

	for _, item := range response.Entries {
		sb.WriteString(fmt.Sprintf("▫️%s\n", item.Entry))

		for _, pronunciation := range item.Pronunctionations {
			if len(pronunciation.Transcriptions) == 0 {
				continue
			}

			sb.WriteString(strings.Join(pronunciation.Context.Regions, ", "))

			if pronunciation.Audio.Url != "" {
				sb.WriteString(fmt.Sprintf(" (<a href=\"%s\">🎧 listen</a>)", pronunciation.Audio.Url))
			}

			sb.WriteString(": ")

			if len(pronunciation.Transcriptions) > 1 {
				sb.WriteRune('\n')
				for _, transcription := range pronunciation.Transcriptions {
					sb.WriteString(fmt.Sprintf("%s [<i>%s</i>]\n", transcription.Transcription, transcription.Notation))
				}
			} else {
				transcription := pronunciation.Transcriptions[0]
				sb.WriteString(fmt.Sprintf("%s [<i>%s</i>]\n", transcription.Transcription, transcription.Notation))
			}
		}

		sb.WriteRune('\n')

		for _, lexeme := range item.Lexemes {
			sb.WriteString(fmt.Sprintf("%s (<i>%s</i>)\n", lexeme.Lemma, lexeme.PartOfSpeech))

			for i, sense := range lexeme.Senses {
				if i >= maxSenses {
					break
				}

				sb.WriteString(fmt.Sprintf("<b>def</b> %s\n", sense.Definition))

				if len(sense.Examples) > 0 {
					for j, example := range sense.Examples {
						if j >= maxExamples {
							break
						}

						sb.WriteString(fmt.Sprintf("<b>ex</b> %s\n", example))
					}
				}

				if len(sense.Antonyms) > 0 {
					sb.WriteString(fmt.Sprintf("<b>ant</b> %s\n", strings.Join(sense.Antonyms, ", ")))
				}

				if len(sense.Synonyms) > 0 {
					sb.WriteString(fmt.Sprintf("<b>syn</b> %s\n", strings.Join(sense.Synonyms, ", ")))
				}

				sb.WriteRune('\n')
			}
		}
	}

	return sb.String(), nil
}
