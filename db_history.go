package main

import (
	"database/sql"
	"fmt"
	"strconv"
)

type DbData struct {
	ID                int
	TimeStamp         string
	PvTotalPower      string
	EmsPower          string
	EmsVoltage        string
	LoadPower         string
	BatteryPercentage string
}

func GetAllDeviceHistory(db *sql.DB) ([]DbData, error) {
	rows, err := db.Query("SELECT id, data_time, pv_input_power_w, battery_power_w, battery_voltage_v, load_power_w, battery_percentage FROM device_data")
	if err != nil {
		return nil, fmt.Errorf("error querying device history: %w", err)
	}
	defer rows.Close()

	var history []DbData
	for rows.Next() {
		var data DbData
		if err := rows.Scan(&data.ID, &data.TimeStamp, &data.PvTotalPower, &data.EmsPower, &data.EmsVoltage, &data.LoadPower, &data.BatteryPercentage); err != nil {
			return nil, fmt.Errorf("error scanning device history: %w", err)
		}
		history = append(history, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through device history rows: %w", err)
	}

	return history, nil
}

// fetch device history data with pagination and date
func GetDeviceHistory(db *sql.DB, dateStr, pageNumStr, pageSizeStr string) ([]DbData, error) {
	pageSize := 10
	pageNum := 1

	// Parse pageSize
	if pageSizeStr != "" {
		size, err := strconv.Atoi(pageSizeStr)
		if err == nil && size > 0 {
			pageSize = size
		}
	}

	// Parse pageNum
	if pageNumStr != "" {
		num, err := strconv.Atoi(pageNumStr)
		if err == nil && num > 0 {
			pageNum = num
		}
	}

	//calculate OFFSET for pagination
	offset := (pageNum - 1) * pageSize

	// Construct the SQL query
	query := `SELECT id, data_time, pv_input_power_w, battery_power_w, battery_voltage_v, load_power_w, battery_percentage FROM device_data`
	var args []interface{}

	// Add date filtering if dateStr is provided
	if dateStr != "" {
		query += ` WHERE strftime('%Y-%m-%d', data_time) = ?`
		args = append(args, dateStr)
	}

	//add limit and offset for pagination
	query += `LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying device history: %w", err)
	}
	defer rows.Close()

	var history []DbData
	for rows.Next() {
		var data DbData
		if err := rows.Scan(&data.ID, &data.TimeStamp, &data.PvTotalPower, &data.EmsPower, &data.EmsVoltage, &data.LoadPower, &data.BatteryPercentage); err != nil {
			return nil, fmt.Errorf("error scanning device history row: %w", err)
		}
		history = append(history, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through device history rows: %w", err)
	}

	return history, nil
}
