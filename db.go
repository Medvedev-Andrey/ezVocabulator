package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func ensureTrainingTableExists(db *sql.DB) error {
	log.Print("Checking Training table exists")

	createTableStatement := `
		CREATE TABLE IF NOT EXISTS training
		( 
			user_id int NOT NULL,
			date timestamp NOT NULL,
			data text
		)`

	_, err := db.Exec(createTableStatement)
	return err
}

func storeTrainingData(db *sql.DB, userID int, data *trainingData) error {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed storing training data for %d user ID. %s", userID, err)
		}
	}()

	insertRowStatement := `
			INSERT INTO training (user_id, date, data)
			VALUES ($1, $2, $3)`

	var jsonData []byte
	jsonData, err = json.Marshal(data)
	if err != nil {
		return err
	}

	date := time.Now().AddDate(0, 0, trainingIterationToDays(data.Iteration))
	_, err = db.Exec(insertRowStatement, userID, date, jsonData)
	if err != nil {
		return err
	}

	return nil
}

func countUserTrainingData(db *sql.DB, userID int) (int, error) {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed requesting training data count for user with ID %d. %s", userID, err)
		}
	}()

	getUserDataStatement := `
		SELECT COUNT(*) FROM training 
		WHERE user_id = $1`
	var count int
	err = db.QueryRow(getUserDataStatement, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func getUserDataToTrain(db *sql.DB, userID int, count int) ([]trainingData, error) {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed requesting training data for user with ID %d. %s", userID, err)
		}
	}()

	if count <= 0 {
		err = fmt.Errorf("training data count to acquire has to be more then zero")
		return nil, err
	}

	getUserTrainingData := `
		SELECT data FROM training 
		WHERE user_id = $1
		ORDER BY date
		LIMIT $2`
	var rows *sql.Rows
	rows, err = db.Query(getUserTrainingData, userID, count)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	defer rows.Close()
	var userDataToTrain []trainingData

	if err == sql.ErrNoRows {
		log.Printf("Found no data to train for user with ID %d", userID)
		err = nil
		return userDataToTrain, nil
	}

	for rows.Next() {
		var rawData []byte
		err = rows.Scan(&rawData)

		if err != nil {
			log.Printf("Error while acquiring user training data from row. %q", err)
			continue
		} else if len(rawData) == 0 {
			log.Printf("Empty user training data for %d", userID)
			continue
		}

		data := trainingData{}
		err = json.Unmarshal(rawData, &data)
		if err != nil {
			log.Printf("Error while deserializing user training data from row. %q", err)
			continue
		}

		userDataToTrain = append(userDataToTrain, data)
	}

	return userDataToTrain, nil
}
