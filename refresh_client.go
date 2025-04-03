package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

//const baseURL = "https://shine-api.felicitysolar.com" // Base URL from documentation

// type RefreshTokenRequest struct {
// 	RefreshToken string `json:"refreshToken"`
// }

// type RefreshTokenResponse struct {
// 	Code    int    `json:"code"`
// 	Message string `json:"message"`
// 	Data    struct {
// 		Token              string `json:"token"`
// 		TokenExpireTime    string `json:"tokenExpireTime"`
// 		RefreshToken       string `json:"refreshToken"`
// 		RefTokenExpireTime string `json:"refTokenExpireTime"`
// 	} `json:"data"`
// }

func RefreshAccessToken() (newAccessToken, newRefreshToken string, newAccessTokenExpiry, newRefreshTokenExpiry time.Time, err error) {
	apiURL, err := url.Parse(baseURL + "/openApi/sec/refreshToken")
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error parsing refresh token URL: %w", err)
	}

	storedTokens, err := loadTokensFromFile()
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error loading tokens for refresh: %w", err)
	}
	if storedTokens == nil || storedTokens.RefreshToken == "" {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("no refresh token available")
	}

	refreshToken := storedTokens.RefreshToken

	requestBody := map[string]string{
		"refreshToken": refreshToken,
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error marshaling refresh request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL.String(), bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error creating refresh token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error sending refresh token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("refresh token request failed with status code: %d", resp.StatusCode)
	}

	var refreshResponse LoginResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&refreshResponse); err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error decoding refresh response body: %w", err)
	}

	if refreshResponse.Code == 998 {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("refresh token expired, please log in again")
	}

	if refreshResponse.Code != 200 {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("refresh token API error: code=%d, message=%s", refreshResponse.Code, refreshResponse.Message)
	}

	// Convert timestamps (milliseconds as strings) to time.Time
	accessTokenExpiryTimestamp, err := strconv.ParseInt(refreshResponse.Data.TokenExpireTime, 10, 64)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error parsing access token expiry time: %w", err)
	}
	newAccessTokenExpiry = time.Unix(accessTokenExpiryTimestamp/1000, 0) // Convert milliseconds to seconds

	refreshTokenExpiryTimestamp, err := strconv.ParseInt(refreshResponse.Data.RefTokenExpireTime, 10, 64)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("error parsing refresh token expiry time: %w", err)
	}
	newRefreshTokenExpiry = time.Unix(refreshTokenExpiryTimestamp/1000, 0) // Convert milliseconds to seconds

	return refreshResponse.Data.AccessToken, refreshResponse.Data.RefreshToken, newAccessTokenExpiry, newRefreshTokenExpiry, nil
}

func performLogin(username, password string) error {
	loginResponse, err := Login(username, password)
	if err != nil {
		fmt.Println("Login Error:", err)
		return err
	}

	fmt.Println("Login Successful!")

	accessTokenExpiryTimestamp, err := strconv.ParseInt(loginResponse.Data.TokenExpireTime, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing access token expiry time: %w", err)
	}
	accessTokenExpiry := time.Unix(accessTokenExpiryTimestamp/1000, 0)

	refreshTokenExpiryTimestamp, err := strconv.ParseInt(loginResponse.Data.RefTokenExpireTime, 10, 64)
	if err != nil {
		return fmt.Errorf("error parsing refresh token expiry time: %w", err)
	}
	refreshTokenExpiry := time.Unix(refreshTokenExpiryTimestamp/1000, 0)

	tokensToSave := &StoredTokens{
		AccessToken:        loginResponse.Data.AccessToken,
		RefreshToken:       loginResponse.Data.RefreshToken,
		AccessTokenExpiry:  accessTokenExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
	}
	if saveErr := saveTokensToFile(tokensToSave); saveErr != nil {
		fmt.Println("Error saving tokens to file:", saveErr)
		return saveErr
	} else {
		fmt.Println("Tokens saved to file.")
	}
	return nil
}
