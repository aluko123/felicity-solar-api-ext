package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type BatteryCalibrationInput struct {
	Voltage    float64 `json:"voltage" binding:"required"`
	Percentage int     `json:"percentage" binding:"required"`
}

func CalibrateBatteryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input BatteryCalibrationInput

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := createBatteryTable(db) //create table if it doesn't exist
		if err != nil {
			fmt.Println("Error creating data table:", err)
			return
		}

		_, err = db.Exec(
			"INSERT INTO battery_calibration (voltage, percentage) VALUES (?, ?)",
			input.Voltage,
			input.Percentage,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save calibration data: %v", err)})
			return
		}

		records, err := GetCalibrationDataHandler(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching calibration data from database"})
			return
		}
		c.JSON(http.StatusOK, records)
	}
}

// createDataTable create device data
func createBatteryTable(db *sql.DB) error {
	_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS battery_calibration (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					voltage REAL NOT NULL,
					percentage INTEGER NOT NULL,
					timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
					)	
	`)
	if err != nil {
		fmt.Printf("error creating battery table: %v ---\n", err)
		return fmt.Errorf("error creating table: %w", err)
	}
	return nil
}

type CalibrationRecord struct {
	ID         int       `json:"id"`
	Voltage    float64   `json:"voltage"`
	Percentage int       `json:"percentage"`
	Timestamp  time.Time `json:"timestamp"`
}

func GetCalibrationDataHandler(db *sql.DB) ([]CalibrationRecord, error) {
	rows, err := db.Query("SELECT id, voltage, percentage, timestamp FROM battery_calibration")
	if err != nil {
		fmt.Println("Error querying calibration data:", err)
		//c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch calibration data"})
		return nil, fmt.Errorf("error querying calibration data: %w", err)
	}
	defer rows.Close()

	var records []CalibrationRecord
	for rows.Next() {
		var record CalibrationRecord
		if err := rows.Scan(&record.ID, &record.Voltage, &record.Percentage, &record.Timestamp); err != nil {
			fmt.Println("Error scanning calibration data row:", err)
			//c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read calibration data"})
			return nil, fmt.Errorf("error scanning calibration data row: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
		//c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process calibration data"})
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return records, nil
}

func GetCalibrationHistory(db *sql.DB) ([]CalibrationRecord, error) {
	rows, err := db.Query("SELECT id, voltage, percentage FROM battery_calibration")
	if err != nil {
		return nil, fmt.Errorf("error querying calibration history: %w", err)
	}
	defer rows.Close()

	var records []CalibrationRecord
	for rows.Next() {
		var record CalibrationRecord
		if err := rows.Scan(&record.ID, &record.Voltage, &record.Percentage); err != nil {
			return nil, fmt.Errorf("error scanning calibration history: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through device history rows: %w", err)
	}

	return records, nil
}
