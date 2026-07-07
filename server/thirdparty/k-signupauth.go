package thirdparty

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

type kResponse[T any] struct {
	Body T `json:"body"`
}

func HandleKOauthCallback(context *gin.Context) {

	code := context.Query("code")
	if code == "" {
		context.String(400, "Hi :)")
		return
	}

	tokenData, err := requestKToken(code)
	if err != nil {
		context.JSON(500, gin.H{"error": "Error occurred while fetching token"})
		return
	}

	// after getting the token, we should identify the user by making a request to the API with the access token
	// only then we can store the refresh token

	userProfile, gameProfile, err := KGetIIDXProfile(tokenData.UserId, tokenData.AccessToken)
	if err != nil {
		context.JSON(500, gin.H{"error": "Error occurred while fetching IIDX profile"})
		return
	}

	log.Println("Successfully fetched IIDX profile for K user:", userProfile.DJName)

	// save the user
	player := models.Player{
		GameID:       tokenData.UserId,
		DJName:       userProfile.DJName,
		DanLevel:     gameProfile.SPDanLevel,
		RefreshToken: sql.NullString{String: tokenData.AccessToken, Valid: true},
		Server:       "K",
	}

	db.DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&player)

	context.Redirect(302, "/success?dj_name="+userProfile.DJName)
}

type kTokenRequest struct {
	ClientId     string `json:"client_id"`
	RedirectUri  string `json:"redirect_uri"`
	Code         string `json:"code"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

type kTokenData struct {
	AccessToken string          `json:"token"`
	Permissions map[string]bool `json:"permissions"`
	UserId      int             `json:"userID"`
}

func requestKToken(code string) (kTokenData, error) {

	requestData := kTokenRequest{
		ClientId:     config.ServerConfig.GetApiConfigByName("K").ClientId,
		RedirectUri:  config.ServerConfig.GetApiConfigByName("K").OauthRedirectUrl,
		Code:         code,
		ClientSecret: config.ServerConfig.GetApiConfigByName("K").Secret,
		GrantType:    "authorization_code",
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Println("Error occurred while marshaling token request: ", err)
		return kTokenData{}, err
	}

	response, err := http.Post(config.ServerConfig.GetApiConfigByName("K").ApiBaseUrl+"/api/v1/oauth/token", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Println("Error occurred while fetching token: ", err)
		return kTokenData{}, err
	}

	defer response.Body.Close()

	tokenResponse, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 {
		log.Println("Error occurred while reading token response: ", string(tokenResponse), " | Status Code: ", response.StatusCode)
		return kTokenData{}, &UnauthorizedError{}
	}

	// unmarshal the token response into a struct
	var data kResponse[kTokenData]
	err = json.Unmarshal(tokenResponse, &data)
	if err != nil {
		log.Println("Error occurred while unmarshaling token response: ", err)
		return kTokenData{}, err
	}

	return data.Body, nil
}
