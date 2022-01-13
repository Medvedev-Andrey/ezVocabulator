package main

import (
	"fmt"
	"math/rand"
	"strings"
)

type responseContent struct {
	content      string
	storeQueries map[string]trainingData
}

type responseBuilder struct {
	contents    []responseContent
	lastContent responseContent
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

				dataToTrain := trainingData{
					Iteration: 1,
					Item:      lexeme.lemma,
					ItemData:  sense,
				}

				storeQuery := generateStoreLexemeDefinitionQuery()
				builder.appendWithQuery(&sb, fmt.Sprintf("\n<b>def</b> %s 📥 %s\n", sense.Definition, storeQuery), storeQuery, dataToTrain)

				if len(sense.Antonyms) > 0 {
					builder.append(&sb, fmt.Sprintf("<b>ant</b> %s\n", strings.Join(sense.Antonyms, ", ")))
				}

				if len(sense.Synonyms) > 0 {
					builder.append(&sb, fmt.Sprintf("<b>syn</b> %s\n", strings.Join(sense.Synonyms, ", ")))
				}

				for j, example := range sense.Examples {
					if j >= maxExamples {
						break
					}

					builder.append(&sb, fmt.Sprintf("<b>ex</b> %s\n", example))
				}
			}

			builder.append(&sb, "\n")
		}
	}

	builder.finish(&sb)
	return builder.contents
}

func generateStoreLexemeDefinitionQuery() string {
	return fmt.Sprintf("%s_%s", StoreTrainingDataPrefix, randStringBytes(6))
}

const strContent = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = strContent[rand.Intn(len(strContent))]
	}

	return string(b)
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

func (builder *responseBuilder) appendWithQuery(strBuilder *strings.Builder, contentPart string, query string, data trainingData) {
	builder.append(strBuilder, contentPart)

	if builder.lastContent.storeQueries == nil {
		builder.lastContent.storeQueries = map[string]trainingData{}
	}

	builder.lastContent.storeQueries[query] = data
}

func (builder *responseBuilder) canAppend(strBuilder *strings.Builder, contentPart string) bool {
	return strBuilder.Len()+len(contentPart) < maxContentLength
}

func (builder *responseBuilder) finish(strBuilder *strings.Builder) {
	builder.lastContent.content = strBuilder.String()
	builder.contents = append(builder.contents, builder.lastContent)
}
