package pages

import (
	"fmt"
	"log"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

func ScoresLower(context *gin.Context) {
	lowerBracketCharts, err := GetBracketCharts("lower")
	if err != nil {
		log.Println("Error occurred while fetching lower bracket charts: ", err)
	}

	lowerBracket := GetBracketScores(lowerBracketCharts)

	// ensure players with higher average rating are sorted for first
	sort.Slice(lowerBracket, func(i, j int) bool {
		return lowerBracket[i].Rating < lowerBracket[j].Rating
	})

	context.HTML(200, "scores.html", gin.H{
		"BracketCharts": lo.Map(lowerBracketCharts, func(chart models.BracketChart, i int) models.ScorePageBracketChart {
			return models.ScorePageBracketChart{
				Title:          chart.Chart.Song.Name,
				TitleLatinized: chart.Chart.Song.NameLatinized,
				ChartLevel:     "SP" + chart.Chart.Difficulty + strconv.Itoa(chart.Chart.Level),
				ChartType:      chart.ChartType,
			}
		}),
		"Bracket": lowerBracket,
	})
}

func ScoresUpper(context *gin.Context) {

	upperBracketCharts, err := GetBracketCharts("upper")
	if err != nil {
		log.Println("Error occurred while fetching upper bracket charts: ", err)
	}

	upperBracket := GetBracketScores(upperBracketCharts)

	// ensure players with higher average rating are sorted for first
	sort.Slice(upperBracket, func(i, j int) bool {
		return upperBracket[i].Rating < upperBracket[j].Rating
	})

	context.HTML(200, "scores.html", gin.H{
		"BracketCharts": lo.Map(upperBracketCharts, func(chart models.BracketChart, i int) models.ScorePageBracketChart {
			return models.ScorePageBracketChart{
				Title:          chart.Chart.Song.Name,
				TitleLatinized: chart.Chart.Song.NameLatinized,
				ChartLevel:     "SP" + chart.Chart.Difficulty + strconv.Itoa(chart.Chart.Level),
				ChartType:      chart.ChartType,
			}
		}),
		"Bracket": upperBracket,
	})
}

type scorePageChartScore struct {
	Player models.Player `json:"player"`
	Score  models.Score  `json:"score"`
}

func GetBracketScores(charts []models.BracketChart) []models.ScorePagePlayerScore {
	if len(charts) == 0 {
		return []models.ScorePagePlayerScore{}
	}

	players, scores := GetBracketPlayersScores(charts)

	if len(players) == 0 {
		log.Println("No players have played yet in this bracket")
		return []models.ScorePagePlayerScore{}
	}

	PlayerIds := lo.KeyBy(players, func(player models.Player) uint {
		return player.ID
	})

	PlayerScores := make(map[models.Player][]models.Score)

	// group scores by player
	lo.ForEach(scores, func(score models.Score, i int) {
		PlayerScores[PlayerIds[score.Player.ID]] = append(PlayerScores[PlayerIds[score.Player.ID]], score)
	})

	// make sure that each player has a score for each chart, if not, add a score with 0 ex and 0 misscount so that the table can be filled correctly
	lo.ForEach(players, func(player models.Player, i int) {
		playerScores := PlayerScores[player]
		for _, chart := range charts {
			if !lo.ContainsBy(playerScores, func(score models.Score) bool {
				return score.BracketChartID == chart.ID
			}) {
				playerScores = append(playerScores, models.Score{
					Player:         player,
					BracketChartID: chart.ID,
					Ex:             0,
					Misscount:      0,
				})
			}
		}
		PlayerScores[player] = playerScores
	})

	frontendPlayerScores := CalculatePerChartRating(charts, PlayerScores)

	var frontendPlayerScoreList []models.ScorePagePlayerScore = make([]models.ScorePagePlayerScore, 0)

	// flatten the map into a list for display in the frontend
	for player, scores := range frontendPlayerScores {
		frontendPlayerScoreList = append(frontendPlayerScoreList, models.ScorePagePlayerScore{
			Player: player,
			Rating: -1, // needs more calculation
			Scores: scores,
		})
	}

	chartsById := lo.Map(charts, func(chart models.BracketChart, index int) uint {
		return chart.ID
	})

	// sort playerScore by chartId in the order of BracketCharts to match the order of charts in the frontend
	for _, playerScore := range frontendPlayerScoreList {
		sort.Slice(playerScore.Scores, func(i, j int) bool {
			chartIdI := playerScore.Scores[i].ChartId
			chartIdJ := playerScore.Scores[j].ChartId
			chartIndexI := lo.IndexOf(chartsById, chartIdI)
			chartIndexJ := lo.IndexOf(chartsById, chartIdJ)
			return chartIndexI < chartIndexJ
		})
	}

	// calculate the total rating for each player based on the ratings of their scores and the weight of each chart
	return CalculatePlayerTotalRatings(frontendPlayerScoreList, charts, players, scores)
}

func CalculatePlayerTotalRatings(frontendPlayerScoreList []models.ScorePagePlayerScore, charts []models.BracketChart, players []models.Player, scores []models.Score) []models.ScorePagePlayerScore {

	chartsById := lo.KeyBy(charts, func(chart models.BracketChart) uint {
		return chart.ID
	})

	for i, playerScore := range frontendPlayerScoreList {
		totalRating := 0.0
		totalWeight := 0.0
		for _, score := range playerScore.Scores {
			_, found := lo.Find(charts, func(chart models.BracketChart) bool {
				return chart.ID == score.ChartId
			})

			if !found {
				log.Println("Error: chart not found for score", score)
				continue
			}
			// count amount of scores for this chart where score above 0,
			// divide by total amount of players to get weight for this chart for this player
			weight := GetWeight(len(players), lo.CountBy(scores, func(s models.Score) bool {
				return s.BracketChartID == score.ChartId && s.Ex > 0
			}))

			if chartsById[score.ChartId].ChartType == "boss" {
				weight *= 2 // boss charts are weighted double
			}

			totalRating += score.Rating * weight
			totalWeight += weight
		}
		frontendPlayerScoreList[i].Rating = totalRating / totalWeight
	}

	return frontendPlayerScoreList
}

func CalculatePerChartRating(charts []models.BracketChart, PlayerScores map[models.Player][]models.Score) map[models.Player][]models.FrontendPlayerScore {

	// bracketchartid -> []{player -> score}
	Placements := make(map[uint][]scorePageChartScore)

	chartsById := lo.KeyBy(charts, func(chart models.BracketChart) uint {
		return chart.ID
	})

	for player, scores := range PlayerScores {
		for _, score := range scores {
			Placements[score.BracketChartID] = append(Placements[score.BracketChartID], scorePageChartScore{
				Player: player,
				Score:  score,
			})
		}
	}

	var frontendPlayerScores map[models.Player][]models.FrontendPlayerScore = make(map[models.Player][]models.FrontendPlayerScore, 0)

	for chartId, chartScores := range Placements {
		totalPlayersWithScore := lo.CountBy(chartScores, func(score scorePageChartScore) bool {
			return score.Score.Ex > 0
		})

		for rank, chartScore := range chartScores {
			frontendPlayerScores[chartScore.Player] = append(frontendPlayerScores[chartScore.Player], models.FrontendPlayerScore{
				ChartId:   chartId,
				ExScore:   chartScore.Score.Ex,
				ScoreRate: fmt.Sprintf("%.2f%%", (float64(chartScore.Score.Ex)/float64(chartsById[chartId].Chart.MaxScore.Int32))*100),
				Rating:    GetRating(rank+1, totalPlayersWithScore),
				Misscount: chartScore.Score.Misscount,
				Lamp:      chartScore.Score.Lamp,
				Timestamp: chartScore.Score.Timestamp.Format("2006-01-02 15:04:05"),
			})
		}
	}
	return frontendPlayerScores
}

func GetBracketCharts(bracket string) ([]models.BracketChart, error) {
	pool, err := db.GetCurrentlyActiveChartPool()
	if err != nil {
		log.Println("Error occurred while fetching currently active chart pool: ", err)

		return []models.BracketChart{}, err
	}

	// extre method to include the song data
	charts, err := db.GetPoolChartsScoresPage(pool)
	if err != nil {
		log.Println("Error occurred while fetching pool charts: ", err)

		return []models.BracketChart{}, err
	}

	charts = lo.Filter(charts, func(chart models.BracketChart, i int) bool {
		return chart.BracketType == bracket
	})

	return charts, nil
}

func GetBracketPlayersScores(charts []models.BracketChart) ([]models.Player, []models.Score) {

	var scores []models.Score = make([]models.Score, 0)

	// Player does not have a navigation to scores so gorm doesn't get it.
	// We can just manually map back to a player after the query since the data is already
	err := db.DB.Model(&models.Score{}).
		Joins("Player").
		Where("scores.bracket_chart_id IN ?", lo.Map(charts, func(chart models.BracketChart, i int) uint {
			return chart.ID
		})).
		Scan(&scores).Error

	if err != nil {
		log.Println("Error occurred while fetching players for bracket charts: ", err)
	}

	return lo.UniqBy(lo.Map(scores, func(score models.Score, i int) models.Player {
		return score.Player
	}), func(player models.Player) uint {
		return player.ID
	}), scores
}

func GetRating(rank int, totalPlayersWithScore int) float64 {
	if totalPlayersWithScore == 0 {
		return 0
	}
	return 100 * (1 - float64(rank-1)/float64(totalPlayersWithScore))
}

func GetWeight(totalPlayers int, submittedScores int) float64 {
	if totalPlayers == 0 {
		return 0
	}
	return float64(submittedScores) / float64(totalPlayers)
}
