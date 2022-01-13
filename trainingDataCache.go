package main

import (
	"fmt"
	"time"
)

var (
	queryToTrainingData     map[string]trainingData = map[string]trainingData{}
	lastCacheTouchTimestamp time.Time               = time.Now()
)

func cacheTrainingDataSet(trainingDataSet map[string]trainingData) {
	clearCacheIfNeeded()

	for query, trainingData := range trainingDataSet {
		queryToTrainingData[query] = trainingData
	}
}

func getTrainingData(query string) (*trainingData, error) {
	trainingData, ok := queryToTrainingData[query]
	if !ok {
		return nil, fmt.Errorf("training data for '%s' query is empty", query)
	}

	delete(queryToTrainingData, query)
	return &trainingData, nil
}

func clearCacheIfNeeded() {
	if time.Since(lastCacheTouchTimestamp).Hours() > queryCacheHoursLifeSpan {
		for k := range queryToTrainingData {
			delete(queryToTrainingData, k)
		}
	}

	lastCacheTouchTimestamp = time.Now()
}
