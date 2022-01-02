package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type dictRequestsRow struct {
	date   string
	userID int
	data   string
}

func ensureDictionaryRequestDBExists(db *sql.DB) error {
	log.Print("Checking dictionary requests database exists")

	createTableStatement := `
		CREATE TABLE IF NOT EXISTS dict_requests 
		( 
			date char(10) NOT NULL, 
			user_id int NOT NULL, 
			data text, 
			PRIMARY KEY (date, user_id)
		)`

	_, err := db.Exec(createTableStatement)
	return err
}

func storeDictionaryRequest(db *sql.DB, userID int, item string) error {
	daysCount := 1
	date := time.Now().AddDate(0, 0, daysCount).Format("2006-01-02")

	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed storing dictionary request for '%d' user ID by %s. %s", userID, date, err)
		} else {
			log.Printf("Successfully stored dictionary request for '%d' user ID by %s", userID, date)
		}
	}()

	log.Printf("Storing dictionary request for '%d' user ID by %s ...", userID, date)
	getRowStatement := `
		SELECT * FROM dict_requests 
		WHERE user_id = $1 AND date = $2`
	row := db.QueryRow(getRowStatement, userID, date)
	err = row.Err()
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err != sql.ErrNoRows {
		dictRequest := new(dictRequestsRow)
		row.Scan(&dictRequest.date, &dictRequest.userID, &dictRequest.data)

		if containsDuplicate(dictRequest.data, item) {
			return nil
		}

		dictRequest.data += formatDictionaryRequest(item, daysCount)
		updateRowStatement := `
			UPDATE dict_requests
			SET data = $1
			WHERE user_id = $2 AND date = $3`
		_, err = db.Exec(updateRowStatement, dictRequest.data, userID, date)
		if err != nil {
			return err
		}
	} else {
		insertRowStatement := `
			INSERT INTO dict_requests (date, user_id, data)
			VALUES ($1, $2, $3)`
		row = db.QueryRow(insertRowStatement, date, userID, formatDictionaryRequest(item, daysCount))
		err = row.Err()
		if err != nil {
			return err
		}
	}

	return nil
}

func formatDictionaryRequest(item string, daysCount int) string {
	return fmt.Sprintf("^%s,%d", item, daysCount)
}

func containsDuplicate(data string, item string) bool {
	return strings.Contains(data, fmt.Sprintf("^%s,", item))
}
