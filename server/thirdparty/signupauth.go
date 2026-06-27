package thirdparty

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

type tokenData struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func HandleOauthCallback(context *gin.Context) {

	code := context.Query("code")
	if code == "" {
		context.String(400, "Hi :)")
		return
	}

	tokenData, err := requestToken(code)
	if err != nil {
		context.JSON(500, gin.H{"error": "Error occurred while fetching token"})
		return
	}

	// after getting the token, we should identify the user by making a request to the API with the access token
	// only then we can store the refresh token

	iidxProfile, err := GetIIDXProfile(tokenData.AccessToken)
	if err != nil {
		context.JSON(500, gin.H{"error": "Error occurred while fetching IIDX profile"})
		return
	}

	log.Println("Successfully fetched IIDX profile for user:", iidxProfile)

	// save the user
	player := models.Player{
		GameID:       iidxProfile.GameID,
		DJName:       iidxProfile.DJName,
		DanLevel:     iidxProfile.SPDanLevel,
		RefreshToken: sql.NullString{String: tokenData.RefreshToken, Valid: true},
	}

	db.DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&player)

	context.Redirect(302, "/success?dj_name="+iidxProfile.DJName)
}

func requestToken(code string) (tokenData, error) {
	queryParams := url.Values{}
	queryParams.Set("grant_type", "authorization_code")
	queryParams.Set("code", code)
	queryParams.Set("redirect_uri", config.ServerConfig.OauthRedirectUrl)
	queryParams.Set("client_id", config.ServerConfig.OauthClientId)
	queryParams.Set("client_secret", config.ServerConfig.OauthSecret)

	queryString := queryParams.Encode()

	response, err := http.Post(config.ServerConfig.ApiBaseUrl+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(queryString))
	if err != nil {
		log.Println("Error occurred while fetching token: ", err)
		return tokenData{}, err
	}

	defer response.Body.Close()

	tokenResponse, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 {
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
