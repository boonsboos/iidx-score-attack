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

// returns true if player is allowed to submit scores to the bracket chart, false otherwise
func checkPlayerPlayingInCorrectBracket(player models.Player, matchingBracketChart models.BracketChart, activeBracketCharts []models.BracketChart) bool {

	legacyBracketCheck := legacyCheck(player, matchingBracketChart, activeBracketCharts)
	if legacyBracketCheck == false {
		return false
	}

	// 5th dan+ players are not allowed in beginner bracket (lv 3-5)
	if player.DanLevel >= 11 && matchingBracketChart.BracketType == "beginner" {
		return genericBracketCheck("beginner", 11, player, matchingBracketChart, activeBracketCharts)
	}

	// 9th dan+ players are not allowed in normal bracket (lv 7-9)
	if player.DanLevel >= 15 && matchingBracketChart.BracketType == "normal" {
		return genericBracketCheck("normal", 15, player, matchingBracketChart, activeBracketCharts)
	}

	// kaiden players are not allowed in hyper bracket (lv 10-12)
	if player.DanLevel == 18 && matchingBracketChart.BracketType == "hyper" {
		return genericBracketCheck("hyper", 18, player, matchingBracketChart, activeBracketCharts)
	}

	return true
}

// ban players that are 6 dan or higher from submitting scores to the lower bracket
// unless they already have scores in the bracket (e.g. they were 6 dan when they submitted earlier scores, and then made it to 7 dan)
// ban players that are 9 dan or higher from submitting scores to the lower and upper bracket
// unless they already have scores in the bracket (e.g. they were 8 dan when they submitted earlier scores, and then made it to 9 dan)
func legacyCheck(player models.Player, matchingBracketChart models.BracketChart, activeBracketCharts []models.BracketChart) bool {
	if player.DanLevel >= 12 && matchingBracketChart.BracketType == "lower" {
		return genericBracketCheck("lower", 12, player, matchingBracketChart, activeBracketCharts)
	}

	if player.DanLevel >= 15 && matchingBracketChart.BracketType == "upper" {
		return genericBracketCheck("upper", 15, player, matchingBracketChart, activeBracketCharts)
	}
	return true
}

func genericBracketCheck(bracket string, threshold int, player models.Player, matchingBracketChart models.BracketChart, activeBracketCharts []models.BracketChart) bool {
	existingScores, err := gorm.G[models.Score](db.DB).
		Where("player_id = ? AND bracket_chart_id in ?",
			player.ID,
			lo.FilterMap(activeBracketCharts, func(chart models.BracketChart, idx int) (uint, bool) {
				return chart.ID, chart.BracketType == bracket
			})).
		Count(db.DefaultTimeout(), "*")

	if err != nil {
		// TODO: notify maintainer
		log.Println("Failed to determine if recently,", models.DanStringsLatin[threshold], "or better", player.Server, "player", player.GameID, "has a score in the", bracket, "bracket", err)
		return false
	}

	if existingScores == 0 {
		log.Println(player.Server, "Player", player.GameID, "is", models.DanStringsLatin[threshold], "or better and submitted a score to", bracket, "bracket chart", matchingBracketChart.ID, "which is not allowed. Ignoring score.")
		return false
	}

	log.Println(player.Server, "Player", player.GameID, "is", models.DanStringsLatin[threshold], "or better and already has scores on", bracket, "bracket chart", matchingBracketChart.ID, "so they are allowed to keep participating in the", bracket, "bracket.")

	return true
}
