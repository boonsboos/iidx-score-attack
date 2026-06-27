package thirdparty

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/models"
)

func GetIIDXScores(authToken string) ([]models.FScore, error) {
	request, err := http.NewRequest("GET", config.ServerConfig.ApiBaseUrl+"/api/iidx/v2/play_history", nil)
	if err != nil {
		log.Println("Error occurred while creating request for play history: ", err)
		return []models.FScore{}, err
	}
	request.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error occurred while fetching play history: ", err)
		return []models.FScore{}, err
	}

	if response.StatusCode != 200 {
		log.Println("Error occurred while fetching play history => Status Code:", response.StatusCode)
		return []models.FScore{}, &UnauthorizedError{}
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error occurred while reading play history response:", err)
		return []models.FScore{}, err
	}

	var scoresResponse struct {
		Scores []models.FScore `json:"_items"`
	}
	err = json.Unmarshal(body, &scoresResponse)
	if err != nil {
		log.Println("Error occurred while deserializing Play history response:", err, " | Response Body:", string(body))
		return []models.FScore{}, err
	}

	return scoresResponse.Scores, nil
}

func GetIIDXProfile(accessToken string) (models.FPlayer, error) {
	request, err := http.NewRequest("GET", config.ServerConfig.ApiBaseUrl+"/api/iidx/v2/player_profile", nil)
	if err != nil {
		log.Println("Error occurred while creating request: ", err)
		return models.FPlayer{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error occurred while fetching IIDX profile: ", err)
		return models.FPlayer{}, err
	}

	if response.StatusCode != 200 {
		log.Println("Error occurred while fetching IIDX profile => Status Code:", response.StatusCode)

		if response.StatusCode == 401 {
			return models.FPlayer{}, &UnauthorizedError{}
		}

		return models.FPlayer{}, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error occurred while reading IIDX profile response:", err)
		return models.FPlayer{}, err
	}

	var profile models.FPlayer
	err = json.Unmarshal(body, &profile)
	if err != nil {
		log.Println("Error occurred while deserializing IIDX profile response:", err, " | Response Body:", string(body))
		return models.FPlayer{}, err
	}

	return profile, nil
}
