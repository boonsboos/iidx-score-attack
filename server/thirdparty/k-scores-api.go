package thirdparty

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/samber/lo"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/models"
)

func KGetIIDXScores(userId int, authToken string) ([]models.KChartScore, error) {
	request, err := http.NewRequest("GET", config.ServerConfig.GetApiConfigByName("K").ApiBaseUrl+"/api/v1/users/"+strconv.Itoa(userId)+"/games/iidx-sp/scores/recent", nil)
	if err != nil {
		log.Println("Error occurred while creating request for play history: ", err)
		return []models.KChartScore{}, err
	}
	request.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error occurred while fetching play history: ", err)
		return []models.KChartScore{}, err
	}

	if response.StatusCode != 200 {
		log.Println("Error occurred while fetching play history => Status Code:", response.StatusCode)
		return []models.KChartScore{}, &UnauthorizedError{}
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error occurred while reading play history response:", err)
		return []models.KChartScore{}, err
	}

	var scoresResponse kResponse[struct {
		Scores []models.KScoreRaw `json:"scores"`
		Charts []models.KChart    `json:"charts"`
	}]
	err = json.Unmarshal(body, &scoresResponse)
	if err != nil {
		log.Println("Error occurred while deserializing Play history response:", err, " | Response Body:", string(body))
		return []models.KChartScore{}, err
	}

	// flatten the scores and charts into a single slice of KChartScore
	scores := make([]models.KChartScore, 0, len(scoresResponse.Body.Scores))
	for _, score := range scoresResponse.Body.Scores {
		chart, found := lo.Find(scoresResponse.Body.Charts, func(c models.KChart) bool {
			return c.ChartId == score.ChartId
		})

		if !found {
			log.Println("Chart not found for score:", score.ScoreId)
			continue
		}

		flattenedScore := score.Flatten()
		flattenedChartScore := chart.Flatten(flattenedScore)
		scores = append(scores, flattenedChartScore)
	}

	return scores, nil
}

func KGetIIDXProfile(userId int, accessToken string) (models.KUser, models.KGameProfile, error) {
	profile, err := getKGameProfile(userId, accessToken)
	if err != nil {
		return models.KUser{}, models.KGameProfile{}, err
	}

	user, err := getKUserProfile(accessToken)
	if err != nil {
		return models.KUser{}, models.KGameProfile{}, err
	}

	return user, profile, nil
}

func getKGameProfile(userId int, accessToken string) (models.KGameProfile, error) {
	request, err := http.NewRequest("GET", config.ServerConfig.GetApiConfigByName("K").ApiBaseUrl+"/api/v1/users/"+strconv.Itoa(userId)+"/games/iidx-sp", nil)
	if err != nil {
		log.Println("Error occurred while creating request: ", err)
		return models.KGameProfile{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error occurred while fetching IIDX profile for K user", userId, ":", err)
		return models.KGameProfile{}, err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Println("Error occurred while fetching IIDX profile for K user", userId, "=> Status Code:", response.StatusCode)

		if response.StatusCode == 401 {
			return models.KGameProfile{}, &UnauthorizedError{}
		}

		body, _ := io.ReadAll(response.Body)
		log.Println("Error occurred while fetching IIDX profile for K user", userId, "=> Response Body:", string(body))
		return models.KGameProfile{}, errors.New(string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error occurred while reading IIDX profile response:", err)
		return models.KGameProfile{}, err
	}

	var gameProfile kResponse[models.KGameProfileRaw]
	err = json.Unmarshal(body, &gameProfile)
	if err != nil {
		log.Println("Error occurred while deserializing IIDX profile response:", err, " | Response Body:", string(body))
		return models.KGameProfile{}, err
	}

	return gameProfile.Body.Flatten(), nil
}

func getKUserProfile(accessToken string) (models.KUser, error) {
	request, err := http.NewRequest("GET", config.ServerConfig.GetApiConfigByName("K").ApiBaseUrl+"/api/v1/users/me", nil)
	if err != nil {
		log.Println("Error occurred while creating request: ", err)
		return models.KUser{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error occurred while fetching K profile: ", err)
		return models.KUser{}, err
	}

	if response.StatusCode != 200 {
		log.Println("Error occurred while fetching K profile => Status Code:", response.StatusCode)

		if response.StatusCode == 401 {
			return models.KUser{}, &UnauthorizedError{}
		}

		return models.KUser{}, errors.New(strconv.Itoa(response.StatusCode))
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error occurred while reading K user response:", err)
		return models.KUser{}, err
	}

	var user kResponse[models.KUser]
	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Println("Error occurred while deserializing K user response:", err, " | Response Body:", string(body))
		return models.KUser{}, err
	}

	return user.Body, nil
}
