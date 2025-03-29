package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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

type StoredTokens struct {
	AccessToken        string    `json:"accessToken"`
	RefreshToken       string    `json:"refreshToken"`
	AccessTokenExpiry  time.Time `json:"accessTokenExpiry"`
	RefreshTokenExpiry time.Time `json:"refreshTokenExpiry"`
}

// response for API errors
type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` //could be string or object
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

// func fetchDataAndStore() error {

// }

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

	//open db
	db, err := sql.Open("sqlite3", "device_data.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	router := gin.Default()

	// serve static frontend files (HTML, CSS, JS)
	router.Static("/static", "./static")

	// Serve the index.html file when the root path is accessed
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	router.POST("api/run_main", func(c *gin.Context) {
		//attempt to load tokens from file on startup
		storedTokens, err := loadTokensFromFile()
		if err != nil {
			fmt.Println("Error loading tokens from file:", err)
		}

		if storedTokens == nil || storedTokens.AccessToken == "" {
			// no tokens in file or loading failed -login then
			err := performLogin(username, password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Initial login failed: %v", err)})
				return
			}
			storedTokens, _ = loadTokensFromFile()
			if storedTokens == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load tokens after initial login"})
			}

		}
		accessToken := storedTokens.AccessToken // Assign values if loaded from file
		//refreshToken := storedTokens.RefreshToken             // Assign values if loaded from file
		accessTokenExpiry := storedTokens.AccessTokenExpiry   // Assign expiry times if loaded from file
		refreshTokenExpiry := storedTokens.RefreshTokenExpiry // Assign expiry times if loaded from file

		//check if access token is expired or about to expire
		if time.Now().Add(time.Minute).After(accessTokenExpiry) {
			//check if refresh token is also expired
			if time.Now().After(refreshTokenExpiry) {
				fmt.Println("Access token and refresh token are expired. Please login again.")
				//clear stored tokens to force new login
				emptyTokens := &StoredTokens{}
				saveTokensToFile(emptyTokens)
				fmt.Println("Attempting to login again...")
				err := performLogin(username, password)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Automatic login failed: %v", err)})
					//handle login failure - maybe exit or retry after a delay
					return
				}
				storedTokens, _ = loadTokensFromFile()
				if storedTokens == nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load tokens after automatic login"})
					return
				}
				accessToken = storedTokens.AccessToken
			}
		} else {

			fmt.Println("Access token expired, attempting refresh.")
			newAccessToken, newRefreshToken, newAccessTokenExpiry, newRefreshTokenExpiry, refreshErr := RefreshAccessToken()
			if refreshErr != nil {
				fmt.Println("Refresh failed:", refreshErr)
				fmt.Println("Attempting to log in again...")
				// Clear stored tokens
				emptyTokens := &StoredTokens{}
				saveTokensToFile(emptyTokens)
				err := performLogin(username, password) // Call a helper function for login
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Automatic login failed after refresh failure: %v", err)})
					// Handle login failure
					return
				}
				storedTokens, _ = loadTokensFromFile()
				if storedTokens == nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load tokens after login after refresh failure"})
					return
				}
				accessToken = storedTokens.AccessToken
			} else {
				fmt.Println("Access token refreshed successfully.")
				//update stored tokens with new access token and maybe new refresh token
				newStoredTokens := &StoredTokens{
					AccessToken:        newAccessToken,
					RefreshToken:       newRefreshToken,
					AccessTokenExpiry:  newAccessTokenExpiry,
					RefreshTokenExpiry: newRefreshTokenExpiry,
				}
				saveTokensToFile(newStoredTokens)
				accessToken = newAccessToken
			}

		}
		//fetch device data history
		fmt.Println("\n--- Fetching Device Data History ---")
		currentTime := time.Now()
		dateStr := currentTime.Format("2006-01-02")
		//dateStr := "2025-03-24"
		//targetTime := time.Date(2025, time.February, 20, 12, 0, 0, 0, time.UTC)
		//dateStr := targetTime.Format("2006-01-02-15:04:05")
		pageNum := "1"
		pageSize := "10"

		_, dataErr := FetchDeviceDataHistory(deviceSN, dateStr, pageNum, pageSize, accessToken)
		if dataErr != nil {
			fmt.Println("Error fetching device data history:", dataErr)
		} else {
			fmt.Println("Device Data History Fetch Successful!")

		}

		history, err := GetAllDeviceHistory(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching data from database"})
			return
		}
		c.JSON(http.StatusOK, history)
	})

	//API endpoint to get history
	router.GET("/api/history", func(c *gin.Context) {
		history, err := GetAllDeviceHistory(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching device from database"})
			return
		}
		c.JSON(http.StatusOK, history)
	})

	//API endpoint to get history with filtering and pagination
	router.GET("/api/history/filtered", func(c *gin.Context) {
		dateStr := c.Query("date")
		pageSizeStr := c.Query("pageSize")
		pageNumStr := c.Query("pageNum")

		history, err := GetDeviceHistory(db, dateStr, pageNumStr, pageSizeStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching filtered data from database"})
			return
		}
		c.JSON(http.StatusOK, history)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" //default
	}

	fmt.Printf("Server listening on port %s...\n", port)
	router.Run(":" + port)
}
