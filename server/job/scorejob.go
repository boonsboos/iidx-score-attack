package job

import (
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
	"iidx.boonsboos.nl/server/thirdparty"
)

// internally cache the auth tokens so we don't have to make refresh calls all the time
var playerTokens map[uint]string = make(map[uint]string)
var workerStartTimer time.Time

func StartWorker() {

	log.Println("Worker configured to run every", config.ServerConfig.WorkerInterval, "seconds")

	workerJob()

	log.Fatalln("Worker stopped processing!")
}

func workerJob() {

	for {
		workerStartTimer = time.Now()

		log.Println("Starting job cycle")

		activeBracketCharts, players, shouldContinue := prepareJob()
		if !shouldContinue {
			time.Sleep(time.Duration(config.ServerConfig.WorkerInterval) * time.Second)
			continue
		}

		waitgroup := sync.WaitGroup{}

		for _, player := range players {
			// waitgroup spawns the task in a goroutine
			waitgroup.Go(func() {
				log.Println("Performing job for player", player.GameID)

				playerJob(player, activeBracketCharts)

				log.Println("Finished job for player", player.GameID)
			})
		}

		// wait for all tasks
		waitgroup.Wait()

		log.Println("Job cycle completed in", time.Since(workerStartTimer).Seconds(), "seconds. Waiting until next cycle...")
		time.Sleep(time.Duration(config.ServerConfig.WorkerInterval)*time.Second - time.Since(workerStartTimer))
	}
}

func prepareJob() ([]models.BracketChart, []models.Player, bool) {
	// check if there is an active chart pool
	activeChartPool, err := db.GetCurrentlyActiveChartPool()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("No active chart pool found - skipping job cycle")
			return nil, nil, false
		}
		log.Panicln("Error occurred while fetching active chart pool:", err)
	}

	// fetch all bracket charts for the active chart pool
	activeBracketCharts, err := db.GetPoolCharts(activeChartPool)
	if err != nil {
		log.Panicln("Error occurred while fetching active bracket charts:", err)
	}

	if len(activeBracketCharts) == 0 {
		log.Println("No active bracket charts found for pool", activeChartPool.ID, activeChartPool.Title, "- skipping job cycle")
		time.Sleep(time.Duration(config.ServerConfig.WorkerInterval) * time.Second)
		return nil, nil, false
	}

	log.Default().Println("Found", len(activeBracketCharts), "active bracket charts for pool", activeChartPool.ID, activeChartPool.Title)

	// fetch all players with a refresh token
	players, err := gorm.G[models.Player](db.DB).Where("refresh_token IS NOT NULL").Find(db.DefaultTimeout())
	if err != nil {
		log.Panicln("Error occurred while fetching users:", err)
	}
	return activeBracketCharts, players, true
}

// fetches the player profile, updates the player in the database if the profile has changed.
// then fetches the player's scores, and updates the scores in the database if they have improved.
func playerJob(player models.Player, activeBracketCharts []models.BracketChart) {

	profile, err := thirdparty.GetIIDXProfile(playerTokens[player.ID])
	if err != nil {
		// try refreshing
		if errors.Is(err, &thirdparty.UnauthorizedError{}) {
			log.Println("Error occurred while fetching profile, going to do an auth refresh for player", player.GameID, "due to:", err)
			retriedProfile, ok := retryFetchingProfile(player)
			if !ok {
				return
			}
			profile = retriedProfile
		} else {
			// TODO: send out an alert to a separate channel where maintainer can see it
			log.Println("Error occurred while fetching profile for player", player.GameID, ":", err)
			return
		}
	}

	log.Println("Fetched profile for player", player.GameID)

	// update the profile if it has changed
	if profile.SPDanLevel > player.DanLevel || profile.DJName != player.DJName {
		log.Println("Updating player", player.GameID, "with new profile data")
		gorm.G[models.Player](db.DB).Where("id = ?", player.ID).Updates(db.DefaultTimeout(), models.Player{
			DJName:   profile.DJName,
			DanLevel: profile.SPDanLevel,
		})
	}

	scores, err := thirdparty.GetIIDXScores(playerTokens[player.ID])
	if err != nil {
		// try refreshing
		if errors.Is(err, &thirdparty.UnauthorizedError{}) {
			log.Println("Error occurred while fetching scores, going to do an auth refresh for player", player.GameID, "due to:", err)
			retriedScores, ok := retryFetchingScores(player)
			if !ok {
				return
			}
			scores = retriedScores
		} else {
			// TODO: send out an alert to a separate channel where maintainer can see it
			log.Println("Error occurred while fetching scores for player", player.GameID, ":", err)
			return
		}
	}

	log.Println("Fetched last", len(scores), "scores for player", player.GameID)

	var scoresSinceLastCycle = lo.Filter(scores, func(score models.FScore, index int) bool {
		return score.Timestamp.After(workerStartTimer.Add(-time.Duration(config.ServerConfig.WorkerInterval)))
	})

	if len(scoresSinceLastCycle) == 0 {
		log.Println("No new scores to process for player", player.GameID)
		return
	}

	log.Println("Found", len(scoresSinceLastCycle), "scores since last cycle for player", player.GameID)

	for _, score := range scoresSinceLastCycle {
		analyzeScore(activeBracketCharts, score, player)
	}
}

func retryFetchingProfile(player models.Player) (models.FPlayer, bool) {
	refreshedToken, err := thirdparty.RefreshAuth(player)
	if err != nil {
		log.Println("Error occurred while refreshing auth for player", player.GameID, ":", err)

		// TODO: send out an alert to a separate channel where maintainer can see it
		return models.FPlayer{}, false
	} else {
		log.Println("Succeeded to refresh auth, going to fetch profile for player", player.GameID, "again")

		// always overwrite the cached token with the refreshed one
		playerTokens[player.ID] = refreshedToken

		profile, err := thirdparty.GetIIDXProfile(refreshedToken)
		if err != nil {
			log.Println("Error occurred while retrying fetching profile for player", player.GameID, ":", err)
			return models.FPlayer{}, false
		}
		return profile, true
	}
}

func retryFetchingScores(player models.Player) ([]models.FScore, bool) {
	refreshedToken, err := thirdparty.RefreshAuth(player)
	if err != nil {
		log.Println("Error occurred while refreshing auth for player", player.GameID, ":", err)

		// TODO: send out an alert to a separate channel where maintainer can see it
		return []models.FScore{}, false
	} else {
		log.Println("Succeeded to refresh auth, going to fetch scores for player", player.GameID, "again")

		// always overwrite the cached token with the refreshed one
		playerTokens[player.ID] = refreshedToken

		scores, err := thirdparty.GetIIDXScores(refreshedToken)
		if err != nil {
			log.Println("Error occurred while retrying fetching scores for player", player.GameID, ":", err)
			return []models.FScore{}, false
		}
		return scores, true
	}
}

func analyzeScore(activeBracketCharts []models.BracketChart, score models.FScore, player models.Player) {
	// find the bracket chart that matches this score
	matchingBracketChart, found := lo.Find(activeBracketCharts, func(bracketChart models.BracketChart) bool {
		return bracketChart.Chart.Difficulty == string(score.Difficulty[0]) &&
			score.PlayStyle == "SINGLE" &&
			bracketChart.Chart.SongId == uint(score.SongId)
	})
	if !found {
		return
	}

	// ban players that are 7 dan or higher from submitting scores to lower bracket
	// we could probably save the score, but that would mean untangling it on the frontend.
	if player.DanLevel >= 13 && matchingBracketChart.BracketType == "lower" {
		log.Println("Player", player.GameID, "is 7 dan+ and submitted a score to the lower bracket chart", matchingBracketChart.ID, "which is not allowed. Ignoring score.")
		return
	}

	log.Println("Processing score for player", player.GameID, "on chart", score.SongId, score.Difficulty)

	// does the player already have a score for this bracket chart?
	existingScore, err := gorm.G[models.Score](db.DB).
		Where("player_id = ? AND bracket_chart_id = ?", player.ID, matchingBracketChart.ID).
		First(db.DefaultTimeout())

	// they do not, create a new score entry
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Println("Updating score for player", player.GameID, "on bracket chart", matchingBracketChart.ID, "with new score", score.ExScore)
		gorm.G[models.Score](db.DB).
			Create(db.DefaultTimeout(), &models.Score{
				PlayerID:       player.ID,
				BracketChartID: matchingBracketChart.ID,
				Ex:             score.ExScore,
				Misscount:      score.MissCount,
				Timestamp:      score.Timestamp,
			})
		return
	}

	if err != nil {
		log.Println("Error occurred while fetching existing score for player", player.GameID, "on bracket chart", matchingBracketChart.ID, ":", err)
		return
	}

	// they do, the new score is higher or misscount is lower, update the existing score entry
	if existingScore.Ex < score.ExScore || existingScore.Misscount > score.MissCount || existingScore.Lamp < score.Lamp {
		log.Println("Updating score for player", player.GameID, "on bracket chart", matchingBracketChart.ID, "with new score", score.ExScore)
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
	}
}
