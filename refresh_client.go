package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

//const baseURL = "https://shine-api.felicitysolar.com" // Base URL from documentation

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshTokenResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token              string `json:"token"`
		TokenExpireTime    string `json:"tokenExpireTime"`
		RefreshToken       string `json:"refreshToken"`
		RefTokenExpireTime string `json:"refTokenExpireTime"`
	} `json:"data"`
}

func RefreshTokenFunc(refreshTokenValue string) (*RefreshTokenResponse, error) {
	apiURL, err := url.Parse(baseURL + "/openApi/sec/refreshToken")
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}

	requestBody := RefreshTokenRequest{
		RefreshToken: refreshTokenValue,
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL.String(), bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
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
		decodeErr := json.NewDecoder(resp.Body).Decode(errorResponse)
		if decodeErr != nil {
			return nil, fmt.Errorf("refresh token request failed with status code: %d, and body decode error: %w", resp.StatusCode, decodeErr)
		}
		return nil, fmt.Errorf("refresh token request failed, status code: %d, message: %s, data: %+v", resp.StatusCode, errorResponse.Message, errorResponse.Data)
	}

	var responseData RefreshTokenResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&responseData); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}
	if responseData.Code != 200 { // Check API-specific error code
		return nil, fmt.Errorf("refresh token API error: code=%d, message=%s", responseData.Code, responseData.Message)
	}

	return &responseData, nil
}

// func main() {
// 	// ... (Get refreshTokenValue somehow - maybe from previous login or storage) ...
// 	refreshTokenValue := "Bearer_eyJhbGciOiJIUzI1NiJ9.eyJpZCI6OTAyMTE1NDA0NDk4ODg5NiwiaWF0IjoxNzE4ODYzODM0fQ._dSC4TGSdeBrK7fbpHcijo23GohZcOK4SMUirToHP9811" // Replace with actual refresh token

// 	tokenResponse, err := RefreshToken(refreshTokenValue)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}

// 	fmt.Println("Refresh Token Success!")
// 	fmt.Printf("Access Token: %s\n", tokenResponse.Data.Token)
// 	fmt.Printf("Refresh Token: %s\n", tokenResponse.Data.RefreshToken)
// 	// ... use tokens ...
// }
