package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	glosbeFmt = "https://glosbe.com/gapi/translate?from={%s}&dest={%s}&format=json&pretty=true&phrase={%s}"
)

type glosbeResponse struct {
	Result string            `json:"result"`
	Tuc    []json.RawMessage `json:"tuc"`
}

type glosbeMeanings struct {
	Items []glosbeItem `json:"meanings"`
}

type glosbeItem struct {
	Text string `json:"text"`
}

func processRequest(request string) ([]byte, error) {
	response, err := http.Get(request)
	if err != nil {
		fmt.Printf("Failed processing request: '%s'", request)
		return nil, err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed processing response body. Request was: '%s'", request)
		return nil, err
	}

	return contents, nil
}

func getMeanings(word string) ([]string, error) {
	contents, err := processRequest(fmt.Sprintf(glosbeFmt, "en", "en", word))
	if err != nil {
		fmt.Printf("Failed getting meanings for '%s'", word)
		return nil, err
	}

	var response glosbeResponse
	err = json.Unmarshal(contents, &response)
	if err != nil || response.Result != "ok" || len(response.Tuc) == 0 {
		fmt.Printf("Failed Glosbe response meanings deserialization for '%s'", word)
		return nil, err
	}

	var meanings glosbeMeanings
	err = json.Unmarshal(response.Tuc[0], &meanings)
	if err != nil {
		fmt.Printf("Failed Glosbe response meanings deserialization for '%s'", word)
		return nil, err
	}

	var result []string
	for _, item := range meanings.Items {
		result = append(result, item.Text)
	}

	return result, nil
}
