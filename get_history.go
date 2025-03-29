package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

// DeviceData struct to represent only fields needed
type DeviceData struct {
	DeviceSn       string `json:"deviceSn"`       // Device serial number
	DeviceDataTime string `json:"deviceDataTime"` // Device data time string
	PvTotalPower   string `json:"pvTotalPower"`   // PV Input Power (W)
	EmsPower       string `json:"emsPower"`       // Battery Power (W)
	EmsVoltage     string `json:"emsVoltage"`     // Battery Voltage (V)
	AcOutputVolt   string `json:"acROutVolt"`     // AC Output Voltage (V)
	AcOutputCurr   string `json:"acROutCurr"`     // AC Output Current (A)
}

// device data history response
type DeviceDataHistoryResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		DataList    []DeviceData `json:"datalist"`
		Total       string       `json:"total"`
		TotalPage   string       `json:"totalPage"`
		PageSize    string       `json:"pageSize"`
		CurrentPage string       `json:"currentPage"`
	} `json:"data"`
}

const dbFileName = "device_data.db"

// fetch historical device data for a single device
func FetchDeviceDataHistory(deviceSn, dateStr, pageNum, pageSize, accessToken string) (*DeviceDataHistoryResponse, error) {
	apiURL, err := url.Parse(baseURL + "/openApi/data/deviceDataHistory/" + deviceSn) //deviceSn added as a parameter
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}

	//fmt.Println(dateStr)

	//add query params
	queryParams := url.Values{}
	queryParams.Add("dateStr", dateStr)
	queryParams.Add("pageNum", pageNum)
	queryParams.Add("pageSize", pageSize)
	apiURL.RawQuery = queryParams.Encode()

	//fmt.Println("Request URL:", apiURL.String())

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	//set authorization header
	req.Header.Set("Authorization", accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorResponse := &ErrorResponse{}
		decodeErr := json.NewDecoder(resp.Body).Decode(errorResponse)
		if decodeErr != nil {
			return nil, fmt.Errorf("data history request failed with status code: %d, and body decode error: %w", resp.StatusCode, decodeErr)
		}
		return nil, fmt.Errorf("data history request failed, status code: %d, message: %s, data: %+v", resp.StatusCode, errorResponse.Message, errorResponse.Data)
	}

	var responseData DeviceDataHistoryResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&responseData); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	if responseData.Code != 200 {
		return nil, fmt.Errorf("data history API error: code=%d, message=%s", responseData.Code, responseData.Message)
	}

	// ---- log to SQLite ----
	db, err := openOrCreateDB() //open or create SQLite database
	if err != nil {
		fmt.Println("Error opening/creating database:", err)
		return &responseData, nil
	}
	defer db.Close()

	err = createDataTable(db) //create table if it doesn't exist
	if err != nil {
		fmt.Println("Error creating data table:", err)
		return &responseData, nil
	}

	err = logDataToDB(db, responseData.Data.DataList) //log data to database
	if err != nil {
		fmt.Println("Error logging data to database:", err)
	}

	return &responseData, nil
}

// openOrCreateDB opens or creates a SQLite db
func openOrCreateDB() (*sql.DB, error) {
	//fmt.Println("--- openOrCreateDB() called ---") // Debug print

	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		fmt.Printf("error opening data table: %v ---\n", err)
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	return db, nil
}

// createDataTable create device data
func createDataTable(db *sql.DB) error {
	_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS device_data (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					device_sn TEXT,
					data_time TEXT,
					pv_input_power_w REAL,
					battery_power_w REAL,
					battery_voltage_v REAL,
					ac_output_voltage REAL,
					ac_output_current REAL,
					load_power_w REAL,
					battery_percentage REAL,
					log_time DATETIME DEFAULT CURRENT_TIMESTAMP
					)	
	`)
	if err != nil {
		fmt.Printf("error creating data table: %v ---\n", err)
		return fmt.Errorf("error creating table: %w", err)
	}
	return nil
}

// log data to DB
func logDataToDB(db *sql.DB, dataList []DeviceData) error {
	tx, err := db.Begin() //start transaction
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback() //rollback if any errors

	stmt, err := tx.Prepare(`
		INSERT INTO device_data(
			device_sn, data_time, pv_input_power_w, battery_power_w, battery_voltage_v, ac_output_voltage, ac_output_current, load_power_w, battery_percentage
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("error preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, data := range dataList {
		load := parseFloat(data.AcOutputCurr) * parseFloat(data.AcOutputVolt)
		battery := (170.0/11.0)*parseFloat(data.EmsVoltage) - (8642.0 / 11.0)
		load_power_w := roundFloat(load, 2)
		battery_percentage := roundFloat(battery, 2)
		_, err := stmt.Exec(
			data.DeviceSn,                 // device_sn TEXT
			data.DeviceDataTime,           // data_time TEXT
			parseFloat(data.PvTotalPower), // pv_input_power_w REAL
			parseFloat(data.EmsPower),     // battery_power_w REAL
			parseFloat(data.EmsVoltage),   // battery_voltage_v REAL
			parseFloat(data.AcOutputVolt), // ac_output_voltage REAL
			parseFloat(data.AcOutputCurr), // ac_output_current REAL
			load_power_w,                  //load_power_w REAL
			battery_percentage,            //battery_percentage REAL
		)
		if err != nil {
			return fmt.Errorf("error inserting data row: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}
	return nil
}

// parseFloat is a helper function to parse string to float64 safely.
func parseFloat(s string) float64 {
	if s == "" {
		return 0.0 // Return 0 if string is empty
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		fmt.Printf("Warning: Could not parse float: %s, error: %v.  Using 0.0\n", s, err)
		return 0.0 // Return 0 on parse error, and log a warning
	}
	return f
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
