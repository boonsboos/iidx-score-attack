package thirdparty

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

func RefreshAuth(player models.Player) (string, error) {
	tokenData, err := refreshToken(player.RefreshToken.String)
	if err != nil {
		log.Println("Error occurred while refreshing auth: ", err)

		player.RefreshToken = sql.NullString{Valid: false}
		db.DB.Save(&player)

		return "", err
	} else {
		// update the player's refresh token
		player.RefreshToken = sql.NullString{String: tokenData.RefreshToken, Valid: true}
		db.DB.Save(&player)

		return tokenData.AccessToken, nil
	}
}

func refreshToken(refreshToken string) (tokenData, error) {
	queryParams := url.Values{}
	queryParams.Set("grant_type", "refresh_token")
	queryParams.Set("refresh_token", refreshToken)
	queryParams.Set("redirect_uri", config.ServerConfig.OauthRedirectUrl)
	queryParams.Set("client_id", config.ServerConfig.OauthClientId)
	queryParams.Set("client_secret", config.ServerConfig.OauthSecret)

	queryString := queryParams.Encode()

	response, err := http.Post(config.ServerConfig.ApiBaseUrl+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(queryString))
	if err != nil {
		log.Println("Error occurred while fetching token: ", err)
		return tokenData{}, err
	}

	if response.StatusCode != 200 {
		log.Println("Error occurred while refreshing token => Status Code: ", response.StatusCode)
		return tokenData{}, err
	}

	defer response.Body.Close()

	tokenResponse, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error occurred while reading token response: ", tokenResponse, " | Status Code: ", response.StatusCode)
		return tokenData{}, err
	}

	// unmarshal the token response into a struct
	var data tokenData
	err = json.Unmarshal(tokenResponse, &data)
	if err != nil {
		log.Println("Error occurred while unmarshaling token response: ", err)
		return tokenData{}, err
	}

	return data, nil
}
