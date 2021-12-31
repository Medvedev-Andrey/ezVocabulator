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
	entries []linguaRobotEntry `json:"entries"`
}

type linguaRobotEntry struct {
	entry             string                      `json:"entry"`
	pronunctionations []linguaRobotPronunciations `json:"pronunciations"`
	lexemes           []linguaRobotLexeme         `json:"lexemes"`
}

type linguaRobotPronunciations struct {
	transcriptions []linguaRobotTranscription `json:"transcriptions"`
	audio          []linguaRobotAudio         `json:"audio"`
	context        linguaRobotContext         `json:"context"`
}

type linguaRobotTranscription struct {
	transcription string `json:"transcription"`
	notation      string `json:"notation"`
}

type linguaRobotAudio struct {
	url string `json:"url"`
}

type linguaRobotContext struct {
	regions []string `json:regions`
}

type linguaRobotLexeme struct {
	lemma        string             `json:"lemma"`
	partOfSpeech string             `json:"partOfSpeech"`
	senses       []linguaRobotSense `json:"senses"`
}

type linguaRobotSense struct {
	definition string   `json:"definition"`
	examples   []string `json:"usageExamples"`
	synonyms   []string `json:"synonyms"`
	antonyms   []string `json:"antonyms"`
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

	if len(response.entries) == 0 {
		sb.WriteString("Nothing has been found ... 😞")
		return "", nil
	}

	for _, item := range response.entries {
		sb.WriteString(fmt.Sprintf("\n▫️%s\n", item.entry))

		for _, pronunciation := range item.pronunctionations {
			regions := strings.Join(pronunciation.context.regions, ", ")
			sb.WriteString(fmt.Sprintf("\n%s: ", regions))

			for _, transcription := range pronunciation.transcriptions {
				sb.WriteString(fmt.Sprintf("%s [<i>%s</i>]\n", transcription.transcription, transcription.notation))
			}
		}

		for _, lexeme := range item.lexemes {
			sb.WriteString(fmt.Sprintf("\n%s (<i>%s</i>)\n", lexeme.lemma, lexeme.partOfSpeech))

			for i, sense := range lexeme.senses {
				if i >= maxSenses {
					break
				}

				sb.WriteString(fmt.Sprintf("\n<code>def</code> %s\n", sense.definition))

				for j, example := range sense.examples {
					if j >= maxExamples {
						break
					}

					sb.WriteString(fmt.Sprintf("<code>ex</code> %s\n", example))
				}

				sb.WriteString(fmt.Sprintf("<code>ant</code> %s\n", strings.Join(sense.antonyms, ", ")))

				sb.WriteString(fmt.Sprintf("<code>syn</code> %s\n", strings.Join(sense.synonyms, ", ")))
			}
		}
	}

	return sb.String(), nil
}
