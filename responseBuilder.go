package main

import (
	"fmt"
	"math/rand"
	"strings"
)

type responseBuilder struct {
	sb           strings.Builder
	storeQueries map[string]trainingData
}

type responseContent struct {
	content      string
	storeQueries map[string]trainingData
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

func (builder *responseBuilder) append(contentPart string) {
	builder.sb.WriteString(contentPart)
}

func (builder *responseBuilder) appendWithQuery(contentPart string, query string, data trainingData) {
	builder.append(contentPart)
	builder.storeQueries[query] = data
}

func (builder *responseBuilder) finish() *responseContent {
	var response responseContent

	response.content = builder.sb.String()
	builder.sb.Reset()

	response.storeQueries = builder.storeQueries
	for k := range builder.storeQueries {
		delete(builder.storeQueries, k)
	}

	return &response
}

func splitResponseContents(responseContent string, partMaxLength int, splitRune rune) []string {
	var responseContentParts []string
	response := []rune(responseContent)

	for len(response) > 0 {
		end := partMaxLength - 1

		if len(response) < partMaxLength {
			end = len(response) - 1
		}

		contentPart := response[:end+1]
		partEnd := findLastRune(contentPart, splitRune)
		if partEnd < 0 {
			partEnd = len(contentPart) - 1
		}

		responseContentParts = append(responseContentParts, string(contentPart[:partEnd+1]))
		response = response[partEnd+1:]
	}

	return responseContentParts
}

func findLastRune(arr []rune, item rune) int {
	for i := len(arr) - 1; i >= 0; i-- {
		if arr[i] == item {
			return i
		}
	}

	return -1
}
