package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

type dictRequestsRow struct {
	date   time.Time
	userID int
	data   string
}

func ensureDictionaryRequestDBExists(db *sql.DB) error {
	log.Print("Checking dictionary requests Database exists")

	createTableStatement := `
		CREATE TABLE IF NOT EXISTS dict_requests 
		( 
			date timestamp NOT NULL, 
			user_id int NOT NULL, 
			data text, 
			PRIMARY KEY (date, user_id)
		)`

	_, err := db.Exec(createTableStatement)
	return err
}

func storeDictionaryRequest(db *sql.DB, userID int, item string) error {
	daysCount := 1
	date := getCurrentDateTimestamp().AddDate(0, 0, daysCount)

	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed storing dictionary request for %d user ID by %s. %s", userID, date, err)
		} else {
			log.Printf("Successfully stored dictionary request for %d user ID by %s", userID, date)
		}
	}()

	var exists bool
	log.Printf("Storing dictionary request for %d user ID by %s ...", userID, date)
	existsStatement := `
		SELECT EXISTS (
			SELECT user_id FROM dict_requests 
			WHERE user_id = $1 AND date = $2 
		)`
	err = db.QueryRow(existsStatement, userID, date).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	log.Printf("Dictionary requests are present for for %d user ID by %s: %t", userID, date, exists)

	if exists {
		getRowStatement := `
			SELECT data FROM dict_requests 
			WHERE user_id = $1 AND date = $2`
		row := db.QueryRow(getRowStatement, userID, date)
		err = row.Err()
		if err != nil {
			return err
		}

		var data string
		err = row.Scan(&data)

		if containsDuplicate(data, item) {
			return nil
		}

		data += formatDictionaryRequest(item, daysCount)
		updateRowStatement := `
			UPDATE dict_requests
			SET data = $1
			WHERE user_id = $2 AND date = $3`

		var res sql.Result
		res, err = db.Exec(updateRowStatement, data, userID, date)
		if err != nil {
			return err
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			err = fmt.Errorf("no rows were affected by Database update")
			return err
		}
	} else {
		insertRowStatement := `
			INSERT INTO dict_requests (date, user_id, data)
			VALUES ($1, $2, $3)`

		var res sql.Result
		res, err = db.Exec(insertRowStatement, date, userID, formatDictionaryRequest(item, daysCount))
		if err != nil {
			return err
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			err = fmt.Errorf("no rows were affected by Database update")
			return err
		}
	}

	return nil
}

func getUserRequests(db *sql.DB, userID int) ([]string, error) {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed requesting history for user with ID %d. %s", userID, err)
		} else {
			log.Printf("Successfully requested history for user with ID %d", userID)
		}
	}()

	log.Printf("Requesting history for user with ID %d ...", userID)
	getUserDataStatement := `
		SELECT data FROM dict_requests 
		WHERE user_id = $1`
	var rows *sql.Rows
	rows, err = db.Query(getUserDataStatement, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	defer rows.Close()

	if err == sql.ErrNoRows {
		log.Printf("Found no history for user with ID %d", userID)
		err = nil
		return []string{}, nil
	}

	var userRequests []string
	re := regexp.MustCompile(`\^([^\^]*),[0-9]+`)
	for rows.Next() {
		var data []byte
		err = rows.Scan(&data)

		if err != nil {
			log.Printf("Error while acquiring user requests history from row. %q", err)
			continue
		} else if len(data) == 0 {
			log.Printf("Empty user requests history for %d", userID)
			continue
		}

		for _, match := range re.FindAllSubmatch(data, -1) {
			userRequests = append(userRequests, string(match[1]))
		}
	}

	return userRequests, nil
}

func getCurrentDateTimestamp() time.Time {
	currentDate := time.Now()
	return time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, currentDate.Location())
}

func formatDictionaryRequest(item string, daysCount int) string {
	return fmt.Sprintf("^%s,%d", item, daysCount)
}

func containsDuplicate(data string, item string) bool {
	return strings.Contains(data, fmt.Sprintf("^%s,", item))
}

type userDictionaryRequest struct {
	item             string
	daysToTrainAfter int
}

func getUserRequestsToTrain(db *sql.DB, userID int, count int) ([]userDictionaryRequest, error) {
	var err error
	defer func() {
		if err != nil {
			log.Printf("Failed requesting words to train for user with ID %d. %s", userID, err)
		} else {
			log.Printf("Successfully requested words to train for user with ID %d", userID)
		}
	}()

	date := getCurrentDateTimestamp()

	log.Printf("Requesting words to train for user with ID %d by %s ...", userID, date)
	getUserDataStatement := `
		SELECT data FROM dict_requests 
		WHERE user_id = $1 AND date <= $2
		ORDER BY date`
	var rows *sql.Rows
	rows, err = db.Query(getUserDataStatement, userID, date)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	defer rows.Close()
	var userDictionaryRequests []userDictionaryRequest

	if err == sql.ErrNoRows {
		log.Printf("Found no words to train for user with ID %d", userID)
		err = nil
		return userDictionaryRequests, nil
	}

	re := regexp.MustCompile(`\^([^\^]*),([0-9]+)`)
	for rows.Next() && len(userDictionaryRequests) < count {
		var data []byte
		err = rows.Scan(&data)

		if err != nil {
			log.Printf("Error while acquiring user requests history from row. %q", err)
			continue
		} else if len(data) == 0 {
			log.Printf("Empty user requests history for %d", userID)
			continue
		}

		for _, match := range re.FindAllSubmatch(data, -1) {
			if len(userDictionaryRequests) >= count {
				break
			}

			newRequest := userDictionaryRequest{
				item:             string(match[1]),
				daysToTrainAfter: int(binary.BigEndian.Uint32(match[2])),
			}

			userDictionaryRequests = append(userDictionaryRequests, newRequest)
		}
	}

	return userDictionaryRequests, nil
}
