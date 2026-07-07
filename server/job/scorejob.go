package job

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
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
				log.Println("Performing job for", player.Server, "player", player.GameID)

				switch player.Server {
				case "F":
					fPlayerJob(player, activeBracketCharts)
				case "K":
					kPlayerJob(player, activeBracketCharts)
				default:
					log.Println("Player with empty server!!!!", player.GameID, "Defaulting to F since it was added first.")
					fPlayerJob(player, activeBracketCharts)
				}

				log.Println("Finished job for", player.Server, "player", player.GameID)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("No active bracket charts found for pool", activeChartPool.ID, activeChartPool.Title, "- skipping job cycle")
			return nil, nil, false
		}

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

// ban players that are 6 dan or higher from submitting scores to the lower bracket
// unless they already have scores in the bracket (e.g. they were 6 dan when they submitted earlier scores, and then made it to 7 dan)
// ban players that are 9 dan or higher from submitting scores to the lower and upper bracket
// unless they already have scores in the bracket (e.g. they were 8 dan when they submitted earlier scores, and then made it to 9 dan)
func checkPlayerPlayingInCorrectBracket(player models.Player, matchingBracketChart models.BracketChart, activeBracketCharts []models.BracketChart) bool {
	if player.DanLevel >= 12 && matchingBracketChart.BracketType == "lower" {
		existingScores, err := gorm.G[models.Score](db.DB).
			Where("player_id = ? AND bracket_chart_id in ?", player.ID,
				lo.FilterMap(activeBracketCharts, func(chart models.BracketChart, idx int) (uint, bool) {
					return chart.ID, chart.BracketType == "lower"
				})).
			Count(db.DefaultTimeout(), "*")

		if err != nil {
			// TODO: notify maintainer
			log.Println(player.Server, "Player", player.GameID, "Failed to determine if a recently 6dan+ player has a score in the lower bracket", err)
			return false
		}

		if existingScores == 0 {
			log.Println(player.Server, "Player", player.GameID, "is 6dan+ and submitted a score to the lower bracket chart", matchingBracketChart.ID, "which is not allowed. Ignoring score.")
			return false
		}

		log.Println(player.Server, "Player", player.GameID, "is 6dan+ and already has scores in the lower bracket chart", matchingBracketChart.ID, "so we will allow them to keep participating in the lower bracket.")
	}

	if player.DanLevel >= 15 && matchingBracketChart.BracketType == "upper" {
		existingScores, err := gorm.G[models.Score](db.DB).
			Where("player_id = ? AND bracket_chart_id in ?", player.ID,
				lo.FilterMap(activeBracketCharts, func(chart models.BracketChart, idx int) (uint, bool) {
					return chart.ID, chart.BracketType == "upper"
				})).
			Count(db.DefaultTimeout(), "*")

		if err != nil {
			// TODO: notify maintainer
			log.Println("Failed to determine if a recently 9dan+ player has a score in the upper bracket", err)
			return false
		}

		if existingScores == 0 {
			log.Println(player.Server, "Player", player.GameID, "is 9dan+ and submitted a score to upper bracket chart", matchingBracketChart.ID, "which is not allowed. Ignoring score.")
			return false
		}

		log.Println(player.Server, "Player", player.GameID, "is 9dan+ and already has scores in the upper bracket so we will allow them to keep participating in the upper bracket.")
	}

	return true
}
