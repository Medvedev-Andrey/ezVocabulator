package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type dictionaryResponse struct {
	original string
	entries  []dictionaryEntry
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
	definitions  []lexemeDefinition
}

type lexemeDefinition struct {
	definition string
	examples   []string
	synonyms   []string
	antonyms   []string
}

const (
	maxContentLength = 4096
)

type response struct {
	contents []string
}

func (response *response) append(builder *strings.Builder, content string) {
	if builder.Len()+len(content) < maxContentLength {
		builder.WriteString(content)
	} else {
		response.contents = append(response.contents, builder.String())
		builder.Reset()
		builder.WriteString(content)
	}
}

func (response *response) finish(builder *strings.Builder) {
	response.contents = append(response.contents, builder.String())
}

func formatUserResponse(dictionaryResponse *dictionaryResponse) []string {
	if len(dictionaryResponse.entries) == 0 {
		return []string{"Nothing has been found ... 😞"}
	}

	var sb strings.Builder
	var response response

	for _, item := range dictionaryResponse.entries {
		response.append(&sb, fmt.Sprintf("▫️%s\n", item.item))

		for _, pronunciation := range item.pronunciations {
			var pronunciationSb strings.Builder

			if len(pronunciation.regions) > 0 {
				pronunciationSb.WriteString(strings.Join(pronunciation.regions, ", "))

				if pronunciation.audioUrl != "" {
					pronunciationSb.WriteString(fmt.Sprintf(" (<a href=\"%s\">🎧 listen</a>)", pronunciation.audioUrl))
				}
			} else if pronunciation.audioUrl != "" {
				pronunciationSb.WriteString(fmt.Sprintf("<a href=\"%s\">🎧 Listen</a>", pronunciation.audioUrl))
			}

			if len(pronunciation.transcriptions) > 0 {
				pronunciationSb.WriteString(fmt.Sprintf(": %s\n", strings.Join(pronunciation.transcriptions, " | ")))
			}

			response.append(&sb, pronunciationSb.String())
		}

		response.append(&sb, "\n")

		for _, lexeme := range item.lexemes {
			response.append(&sb, fmt.Sprintf("%s (<i>%s</i>)\n", lexeme.lemma, lexeme.partOfSpeech))

			for i, sense := range lexeme.definitions {
				if i >= maxSenses {
					break
				}

				response.append(&sb, fmt.Sprintf("\n<b>def</b> %s\n", sense.definition))

				if len(sense.antonyms) > 0 {
					response.append(&sb, fmt.Sprintf("<b>ant</b> %s\n", strings.Join(sense.antonyms, ", ")))
				}

				if len(sense.synonyms) > 0 {
					response.append(&sb, fmt.Sprintf("<b>syn</b> %s\n", strings.Join(sense.synonyms, ", ")))
				}

				if len(sense.examples) > 0 {
					for j, example := range sense.examples {
						if j >= maxExamples {
							break
						}

						response.append(&sb, fmt.Sprintf("<b>ex</b> %s\n", example))
					}
				}
			}

			response.append(&sb, "\n")
		}
	}

	response.finish(&sb)
	return response.contents
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
		fmt.Printf("Failed processing response body'")
		return nil, err
	}

	return contents, nil
}
