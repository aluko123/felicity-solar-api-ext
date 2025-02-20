package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const baseURL = "https://shine-api.felicitysolar.com" // Base URL from documentation
const publicKeyStr = "MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAK0GDivaRzIKeTmQnAxAYh2LChuHWDp0yHZ0zIvm+Eoi7J+rx7phqR7EtkBDO3HWqAXVkNDeeQaU32P5w1Q4FVUCAwEAAQ=="
const tokenFile = "tokens.json"

// login request struct
type LoginRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"` //needs RSA encryption
}

// login succesful response struct
type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// AccessToken        string `json:"data.token"`
	// TokenExpireTime    string `json:"data.tokenExpireTime"`
	// RefreshToken       string `json:"data.refreshToken"`
	// RefTokenExpireTime string `json:"data.refTokenExpireTime"`
	Data struct {
		AccessToken        string `json:"token"`
		TokenExpireTime    string `json:"tokenExpireTime"`
		RefreshToken       string `json:"refreshToken"`
		RefTokenExpireTime string `json:"refTokenExpireTime"`
	} `json:"data"`
}

// response for API errors
type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` //could be string or object
}

type StoredTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func RSAEncrypt(plainText string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		return "", fmt.Errorf("error decoding public key :%w", err)
	}

	pub, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		//try to parse PEM format if X509 parsing fails (though the key is not PEM in this case)
		block, _ := pem.Decode(publicKeyBytes)
		if block != nil {
			if pub, err = x509.ParsePKIXPublicKey(block.Bytes); err == nil {
				// Successfully parsed PEM-encoded key (unlikely in this case, but for robustness)
			} else {
				return "", fmt.Errorf("error parsing public key (X509/PEM): %w", err)
			}
		} else {
			return "", fmt.Errorf("error parsing public key (X509): %w", err)
		}
	}

	rsaPublicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("invalid public key type, not RSA")
	}

	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPublicKey, []byte(plainText))
	if err != nil {
		return "", fmt.Errorf("encryption error: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encryptedBytes), nil

}

func Login(username, password string) (*LoginResponse, error) {
	apiURL, err := url.Parse(baseURL + "/openApi/sec/login")
	if err != nil {
		return nil, fmt.Errorf("error pasrsing URL: %w", err)
	}

	encryptedPassword, err := RSAEncrypt(password)
	if err != nil {
		return nil, fmt.Errorf("password encrpytion error: %w", err)
	}
	//fmt.Printf("%s", encryptedPassword)

	requestBody := LoginRequest{
		UserName: username,
		Password: encryptedPassword,
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL.String(), bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorResponse := &ErrorResponse{}
		decoderErr := json.NewDecoder(resp.Body).Decode(errorResponse)
		if decoderErr != nil {
			return nil, fmt.Errorf("login request failed with status code: %d, and body decode error: %w", resp.StatusCode, decoderErr)
		}
		return nil, fmt.Errorf("login request failed, status code: %d, message: %s, data: %v", resp.StatusCode, errorResponse.Message, errorResponse.Data)
	}

	var responseData LoginResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&responseData); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	if responseData.Code != 200 { // Check API-specific error code
		return nil, fmt.Errorf("login API error: code=%d, message=%s", responseData.Code, responseData.Message)
	}

	//fmt.Println(&responseData)

	return &responseData, nil
}

func saveTokensToFile(tokens *StoredTokens) error {
	tokenJSON, err := json.MarshalIndent(tokens, "", "  ") // Pretty JSON formatting
	if err != nil {
		return fmt.Errorf("error marshaling tokens to JSON: %w", err)
	}
	err = os.WriteFile(tokenFile, tokenJSON, 0600) // 0600: read/write for owner only (secure file permissions)
	if err != nil {
		return fmt.Errorf("error writing tokens to file: %w", err)
	}
	return nil
}

func loadTokensFromFile() (*StoredTokens, error) {
	tokenJSON, err := os.ReadFile(tokenFile)
	if err != nil {
		if os.IsNotExist(err) { // File doesn't exist - that's OK on first run
			return nil, nil // Return nil tokens, no error
		}
		return nil, fmt.Errorf("error reading tokens from file: %w", err)
	}

	var tokens StoredTokens
	err = json.Unmarshal(tokenJSON, &tokens)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling tokens from JSON: %w", err)
	}
	return &tokens, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	deviceSN := os.Getenv("DEVICE_SN")

	if username == "" || password == "" || deviceSN == "" {
		log.Fatal("USERNAME, PASSWORD and DEVICE_SN must be set in .env file") // Ensure variables are set
	}

	//attempt to load tokens from file on startup
	storedTokens, err := loadTokensFromFile()
	if err != nil {
		fmt.Println("Error loading tokens from file:", err)
	}

	var accessToken string
	var refreshToken string

	if storedTokens != nil && storedTokens.AccessToken != "" && storedTokens.RefreshToken != "" {
		accessToken = storedTokens.AccessToken
		refreshToken = storedTokens.RefreshToken
		fmt.Println("Tokens loaded from file.")
	} else {
		// no tokens in file or loading failed -login then
		loginResponse, err := Login(username, password)
		if err != nil {
			fmt.Println("Login Error:", err)
			return
		}

		fmt.Println("Login Successful!")
		//fmt.Println(loginResponse)
		accessToken = loginResponse.Data.AccessToken
		refreshToken = loginResponse.Data.RefreshToken

		//save tokens to file after successful login
		tokensToSave := &StoredTokens{AccessToken: accessToken, RefreshToken: refreshToken}
		if saveErr := saveTokensToFile(tokensToSave); saveErr != nil {
			fmt.Println("Error saving tokens to file:", saveErr)
		} else {
			fmt.Println("Tokens saved to file.")
		}
	}

	fmt.Printf("Access Token: %s\n", accessToken)
	fmt.Printf("Refresh Token: %s\n", refreshToken)

	//fetch device data history
	fmt.Println("\n--- Fetching Device Data History ---")
	currentTime := time.Now()
	dateStr := currentTime.Format("2025-02-19")
	pageNum := "1"
	pageSize := "50"

	_, dataErr := FetchDeviceDataHistory(deviceSN, dateStr, pageNum, pageSize, accessToken)
	if dataErr != nil {
		fmt.Println("Error fetching device data history:", dataErr)
	} else {
		fmt.Println("Device Data History Fetch Successful!")

		// if len(dataHistoryResponse.Data.DataList) > 0 {
		// 	fmt.Println("\n--- Device Data ---")
		// 	fmt.Printf("%-15s %-15s %-20s %-20s %-20s %-20s %-20s %-25s %-35s\n",
		// 		"DeviceSn", "PV Power(kWh)", "DataTime", "PV Input Power(W)", "AC Input Power(W)",
		// 		"Battery Voltage(V)", "Battery Power(W)", "AC Output Power(W)", "AC Apparent Power(VA)") // Shortened header
		// 	fmt.Printf("%-15s %-15s %-20s %-20s  %-20s %-20s %-20s %-25s %-35s\n",
		// 		"---------", "--------------", "-------------------", "-------------------", "-------------------", "-------------------",
		// 		"-------------------", "--------------------------", "-----------------------------------") // Shortened separator

		// 	for _, data := range dataHistoryResponse.Data.DataList {
		// 		fmt.Printf("%-15s %-15s %-20s %-20s %-20s %-20s %-20s %-25s %-35s\n",
		// 			data.DeviceSn,
		// 			data.EpvToday, // PV Power Today (kWh)
		// 			data.DeviceDataTime,
		// 			data.PvTotalPower,        // PV Input Power (W)
		// 			data.AcTtlInpower,        // AC Input Power (W) - Total Grid Power
		// 			data.EmsVoltage,          // Battery Voltage (V)
		// 			data.EmsPower,            // Battery Power (W)
		// 			data.AcTotalOutActPower,  // AC Output Power (W) - Total Backup Load Active Power
		// 			data.AcTotalOutAppaPower, // Total AC Output Apparent Power (VA)
		// 		)
		// 	}
		// } else {
		// 	fmt.Println("\n--- No Device Data Records found in response ---")
		// }
	}

	//fmt.Println("\n--- Ready to make API requests using Access Token ---")
	//fmt.Println("Access Token:", accessToken) // Use this accessToken for subsequent API calls
}
