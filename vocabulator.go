package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

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
