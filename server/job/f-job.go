package job

import (
	"errors"
	"log"
	"math"
	"sort"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
	"iidx.boonsboos.nl/server/thirdparty"
)

// fetches the player profile, updates the player in the database if the profile has changed.
// then fetches the player's scores, and updates the scores in the database if they have improved.
func fPlayerJob(player models.Player, activeBracketCharts []models.BracketChart) {

	profile, err := thirdparty.FGetIIDXProfile(playerTokens[player.ID])
	if err != nil {
		// try refreshing
		if errors.Is(err, &thirdparty.UnauthorizedError{}) {
			log.Println("Error occurred while fetching profile, going to do an auth refresh for", player.Server, "player", player.GameID, "due to:", err)
			retriedProfile, ok := retryFetchingProfile(player)
			if !ok {
				return
			}
			profile = retriedProfile
		} else {
			// TODO: send out an alert to a separate channel where maintainer can see it
			log.Println("Error occurred while fetching profile for", player.Server, "player", player.GameID, ":", err)
			return
		}
	}

	log.Println("Fetched profile for", player.Server, "player", player.GameID)

	// update the profile if it has changed
	if profile.SPDanLevel > player.DanLevel || profile.DJName != player.DJName {
		if profile.SPDanLevel > player.DanLevel {
			log.Println(player.Server, "Player", player.GameID, "has leveled up from", models.DanStringsLatin[player.DanLevel], "to", models.DanStringsLatin[profile.SPDanLevel])
		}

		if profile.DJName != player.DJName {
			log.Println(player.Server, "Player", player.GameID, "has changed their DJ name")
		}

		err := db.DB.Model(&player).UpdateColumns(models.Player{
			DJName:   profile.DJName,
			DanLevel: profile.SPDanLevel,
		}).Error
		if err != nil {
			// TODO: notify maintainer
			log.Println("Error occurred while updating player profile for", player.Server, "player", player.GameID, ":", err)
		}
	}

	scores, err := thirdparty.FGetIIDXScores(playerTokens[player.ID])
	if err != nil {
		// try refreshing
		if errors.Is(err, &thirdparty.UnauthorizedError{}) {
			log.Println("Error occurred while fetching scores, going to do an auth refresh for", player.Server, "player", player.GameID, "due to:", err)
			retriedScores, ok := retryFetchingScores(player)
			if !ok {
				return
			}
			scores = retriedScores
		} else {
			// TODO: send out an alert to a separate channel where maintainer can see it
			log.Println("Error occurred while fetching scores for", player.Server, "player", player.GameID, ":", err)
			return
		}
	}

	// process oldest scores first
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Timestamp.Before(scores[j].Timestamp)
	})

	log.Println("Fetched last", len(scores), "scores for", player.Server, "player", player.GameID)

	poolStartTime, poolEndTime, err := db.GetCurrentlyActiveChartPoolStartTime()
	if err != nil {
		log.Println("Error occurred while fetching active chart pool start time for", player.Server, "player", player.GameID, ":", err)
		return
	}

	var updatedScores int
	for _, score := range scores {
		if score.Timestamp.Before(poolStartTime) || score.Timestamp.After(poolEndTime) {
			continue
		}

		updatedScores += analyzeFScore(activeBracketCharts, score, player)
	}
	log.Println("Updated", updatedScores, "scores for", player.Server, "player", player.GameID)
}

func retryFetchingProfile(player models.Player) (models.FPlayer, bool) {
	refreshedToken, err := thirdparty.RefreshFAuth(player)
	if err != nil {
		log.Println("Error occurred while refreshing auth for player", player.GameID, ":", err)

		// TODO: send out an alert to a separate channel where maintainer can see it
		return models.FPlayer{}, false
	} else {
		log.Println("Succeeded to refresh auth, going to fetch profile for", player.Server, "player", player.GameID, "again")

		// always overwrite the cached token with the refreshed one
		playerTokens[player.ID] = refreshedToken

		profile, err := thirdparty.FGetIIDXProfile(refreshedToken)
		if err != nil {
			log.Println("Error occurred while retrying fetching profile for", player.Server, "player", player.GameID, ":", err)
			return models.FPlayer{}, false
		}
		return profile, true
	}
}

func retryFetchingScores(player models.Player) ([]models.FScore, bool) {
	refreshedToken, err := thirdparty.RefreshFAuth(player)
	if err != nil {
		log.Println("Error occurred while refreshing auth for", player.Server, "player", player.GameID, ":", err)

		// TODO: send out an alert to a separate channel where maintainer can see it
		return []models.FScore{}, false
	} else {
		log.Println("Succeeded to refresh auth, going to fetch scores for", player.Server, "player", player.GameID, "again")

		// always overwrite the cached token with the refreshed one
		playerTokens[player.ID] = refreshedToken

		scores, err := thirdparty.FGetIIDXScores(refreshedToken)
		if err != nil {
			log.Println("Error occurred while retrying fetching scores for", player.Server, "player", player.GameID, ":", err)
			return []models.FScore{}, false
		}
		return scores, true
	}
}

func analyzeFScore(activeBracketCharts []models.BracketChart, score models.FScore, player models.Player) int {
	// find the bracket chart that matches this score
	matchingBracketChart, found := lo.Find(activeBracketCharts, func(bracketChart models.BracketChart) bool {
		return bracketChart.Chart.Difficulty == string(score.Difficulty[0]) &&
			score.PlayStyle == "SINGLE" &&
			bracketChart.Chart.SongId == uint(score.SongId)
	})
	if !found {
		return 0
	}

	playingInCorrectBracket := checkPlayerPlayingInCorrectBracket(player, matchingBracketChart, activeBracketCharts)
	if !playingInCorrectBracket {
		return 0
	}

	log.Println("Processing score for", player.Server, "player", player.GameID, "on chart", score.SongId, score.Difficulty)

	// does the player already have a score for this bracket chart?
	existingScore, err := gorm.G[models.Score](db.DB).
		Where("player_id = ? AND bracket_chart_id = ?", player.ID, matchingBracketChart.ID).
		First(db.DefaultTimeout())

	if err != nil {
		// they do not, create a new score entry
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("Updating score for", player.Server, "player", player.GameID, "on bracket chart", matchingBracketChart.ID, "with new score", score.ExScore)
			gorm.G[models.Score](db.DB).
				Create(db.DefaultTimeout(), &models.Score{
					PlayerID:       player.ID,
					BracketChartID: matchingBracketChart.ID,
					Ex:             score.ExScore,
					Misscount:      score.MissCount,
					Lamp:           score.Lamp,
					Timestamp:      score.Timestamp,
				})
			return 1
		}

		log.Println("Error occurred while fetching existing score for", player.Server, "player", player.GameID, "on bracket chart", matchingBracketChart.ID, ":", err)
		return 0
	}

	// they do, the new score is higher or misscount is lower, update the existing score entry
	if existingScore.Ex < score.ExScore || existingScore.Misscount > score.MissCount || existingScore.Lamp < score.Lamp {
		log.Println("Updating score for", player.Server, "player", player.GameID, "on bracket chart", matchingBracketChart.ID, "with new score", score.ExScore)

		// ignore people quitting out (DEATH = -1 bp) by keeping their existing misscount
		if score.MissCount == -1 {
			score.MissCount = existingScore.Misscount
		}

		// if the existing score has a misscount of -1, it means the player quit out on their previous attempt, so we should keep the new misscount
		if existingScore.Misscount == -1 {
			existingScore.Misscount = score.MissCount
		}

		gorm.G[models.Score](db.DB).
			Where("player_id = ? AND bracket_chart_id = ?", player.ID, matchingBracketChart.ID).
			Updates(db.DefaultTimeout(), models.Score{
				// verify if the ex score is higher than the existing score's ex score, if so, update it as well
				Ex: int(math.Max(float64(score.ExScore), float64(existingScore.Ex))),
				// verify if the misscount is lower than the existing score's misscount, if so, update it as well
				Misscount: int(math.Min(float64(score.MissCount), float64(existingScore.Misscount))),
				Lamp:      int(math.Max(float64(score.Lamp), float64(existingScore.Lamp))),
				Timestamp: score.Timestamp,
			})

		return 1
	}
	return 0
}
