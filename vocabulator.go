package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
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
	definitions  []lexemeDefinition
}

type lexemeDefinition struct {
	definition string
	examples   []string
	synonyms   []string
	antonyms   []string
}

const (
	maxContentLength             = 4096
	StoreDictionaryRequestPrefix = "/slm"
)

type responseContent struct {
	content      string
	storeQueries map[string]lexemeDefinition
}

type responseBuilder struct {
	contents    []responseContent
	lastContent responseContent
}

func (builder *responseBuilder) append(strBuilder *strings.Builder, contentPart string) {
	if !builder.canAppend(strBuilder, contentPart) {
		builder.lastContent.content = strBuilder.String()
		builder.contents = append(builder.contents, builder.lastContent)

		builder.lastContent = responseContent{}
		strBuilder.Reset()
	}

	strBuilder.WriteString(contentPart)
}

func (builder *responseBuilder) appendWithQuery(strBuilder *strings.Builder, contentPart string, query string, defToStore lexemeDefinition) {
	builder.append(strBuilder, contentPart)
	builder.lastContent.storeQueries[query] = defToStore
}

func (builder *responseBuilder) canAppend(strBuilder *strings.Builder, contentPart string) bool {
	return strBuilder.Len()+len(contentPart) < maxContentLength
}

func (builder *responseBuilder) finish(strBuilder *strings.Builder) {
	builder.lastContent.content = strBuilder.String()
	builder.contents = append(builder.contents, builder.lastContent)
}

type formattedDictionaryResponse struct {
	textContents              []string
	commandToLexemeDefinition map[string]lexemeDefinition
}

func formatUserResponse(dictionaryResponse *dictionaryResponse) []responseContent {
	var sb strings.Builder
	var builder responseBuilder

	for _, item := range dictionaryResponse.entries {
		builder.append(&sb, fmt.Sprintf("▫️%s\n", item.item))

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
				pronunciationSb.WriteString(fmt.Sprintf(": %s", strings.Join(pronunciation.transcriptions, " | ")))
			}

			pronunciationSb.WriteRune('\n')
			builder.append(&sb, pronunciationSb.String())
		}

		builder.append(&sb, "\n")

		for _, lexeme := range item.lexemes {
			builder.append(&sb, fmt.Sprintf("<u>%s (<i>%s</i>)</u>\n", lexeme.lemma, lexeme.partOfSpeech))

			for i, sense := range lexeme.definitions {
				if i >= maxSenses {
					break
				}

				storeQuery := generateStoreLexemeDefinitionQuery()
				builder.append(&sb, fmt.Sprintf("\n<b>def</b> %s <i>Store? %s</i>\n", sense.definition, storeQuery))

				if len(sense.antonyms) > 0 {
					builder.append(&sb, fmt.Sprintf("<b>ant</b> %s\n", strings.Join(sense.antonyms, ", ")))
				}

				if len(sense.synonyms) > 0 {
					builder.append(&sb, fmt.Sprintf("<b>syn</b> %s\n", strings.Join(sense.synonyms, ", ")))
				}

				if len(sense.examples) > 0 {
					for j, example := range sense.examples {
						if j >= maxExamples {
							break
						}

						builder.append(&sb, fmt.Sprintf("<b>ex</b> %s\n", example))
					}
				}
			}

			builder.append(&sb, "\n")
		}
	}

	builder.finish(&sb)
	return builder.contents
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

func generateStoreLexemeDefinitionQuery() string {
	return fmt.Sprintf("%s_%s", StoreDictionaryRequestPrefix, RandStringBytes(6))
}

const strContent = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = strContent[rand.Intn(len(strContent))]
	}

	return string(b)
}
