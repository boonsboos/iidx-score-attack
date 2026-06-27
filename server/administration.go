package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

func RegisterMaintenanceRoutes(router *gin.Engine) {
	router.POST("/maintenance/charts", importChartData)

	router.POST("/maintenance/pool", createNewPool)
	router.POST("/maintenance/pool/:id", addChartToPool)

	router.POST("/maintenance/charts/maxscore", updateChartMaxScores)
	// TODO: add route to reset user auth
}

// adminHeader is used to validate that the request is coming from an admin
type adminHeader struct {
	AdminKey string `header:"X-Admin-Key" binding:"required"`
}

type importChart struct {
	SongID      uint             `json:"song_id"`
	Title       string           `json:"title"`
	TitleASCII  string           `json:"title_ascii"`
	Genre       string           `json:"genre"`
	Artist      string           `json:"artist"`
	GameVersion uint             `json:"game_version"`
	SP          difficultySpread `json:"SP"`
	DP          difficultySpread `json:"DP"`
}

type difficultySpread struct {
	B int `json:"B"`
	N int `json:"N"`
	H int `json:"H"`
	A int `json:"A"`
	L int `json:"L"`
}

func checkAdminKey(context *gin.Context) bool {
	var header adminHeader
	if err := context.ShouldBindHeader(&header); err != nil {
		context.Status(400)
		return false
	}

	if header.AdminKey != config.ServerConfig.AdminKey {
		context.Status(400)
		return false
	}
	return true
}

func importChartData(context *gin.Context) {
	if !checkAdminKey(context) {
		return
	}

	body, err := io.ReadAll(context.Request.Body)
	if err != nil {
		log.Println("Error occurred while reading request body: ", err)
		context.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var charts []importChart
	err = json.Unmarshal(body, &charts)
	if err != nil {
		log.Println("Error occurred while unmarshalling chart data: ", err)
		context.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if len(charts) == 0 {
		context.JSON(400, gin.H{"error": "No chart data provided"})
		return
	}

	var songs []models.Song

	chartCount := 0

	for _, chart := range charts {

		var songcharts []models.Chart

		for _, difficulty := range []string{"B", "N", "H", "A", "L"} {

			var level = reflect.ValueOf(chart.SP).FieldByName(difficulty).Int()
			if level == 0 {
				continue
			}

			songcharts = append(songcharts, models.Chart{
				SongId:     chart.SongID,
				Level:      int(level),
				Difficulty: difficulty,
			})

			chartCount++
		}

		song := models.Song{
			ID:            chart.SongID,
			Name:          chart.Title,
			NameLatinized: chart.TitleASCII,
			Artist:        chart.Artist,
			VersionId:     chart.GameVersion,
			Charts:        songcharts,
		}

		songs = append(songs, song)
	}

	// bulk insert songs, including charts due to model associations
	gorm.G[models.Song](db.DB).CreateInBatches(db.DefaultTimeout(), &songs, 100)

	context.JSON(200, gin.H{
		"message": fmt.Sprintf("Successfully imported %d songs and %d charts", len(songs), chartCount),
	})
}

type createPoolRequest struct {
	Title       string    `json:"title"`
	ActiveFrom  time.Time `json:"from"`
	ActiveUntil time.Time `json:"until"`
}

func createNewPool(context *gin.Context) {
	if !checkAdminKey(context) {
		return
	}

	var request createPoolRequest
	err := context.ShouldBindJSON(&request)
	if err != nil {
		context.JSON(400, gin.H{"error": err.Error()})
		return
	}

	pool := models.ChartPool{
		Title:       request.Title,
		ActiveFrom:  request.ActiveFrom,
		ActiveUntil: request.ActiveUntil,
	}

	if !pool.IsValid() {
		context.JSON(400, gin.H{"error": "Pool starts after it ends"})
		return
	}

	overlap, err := gorm.G[models.ChartPool](db.DB).
		Where("active_from < ? AND active_until > ?", pool.ActiveUntil, pool.ActiveFrom).
		First(db.DefaultTimeout())

	if overlap.ID != 0 && err == nil {
		context.JSON(400, gin.H{
			"error":   "Overlapping pool exists",
			"culprit": overlap,
		})
		return
	}

	gorm.G[models.ChartPool](db.DB).Create(db.DefaultTimeout(), &pool)

	context.JSON(200, gin.H{
		"message": "Successfully created new chart pool",
		"pool":    pool,
	})
}

type addBracketChartRequest struct {
	PoolId      uint   `json:"pool_id"`
	ChartId     uint   `json:"chart_id"`
	BracketType string `json:"bracket_type"` // "upper" or "lower"
	ChartType   string `json:"chart_type"`   // "normal" or "boss"
}

func addChartToPool(context *gin.Context) {
	if !checkAdminKey(context) {
		return
	}

	var request addBracketChartRequest
	err := context.ShouldBindJSON(&request)
	if err != nil {
		context.JSON(400, gin.H{"error": err.Error()})
		return
	}

}

func updateChartMaxScores(context *gin.Context) {
	if !checkAdminKey(context) {
		return
	}

	var notecountsBody map[string]map[string]int
	err := context.ShouldBindJSON(&notecountsBody)
	if err != nil {
		context.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var totalUpdatedCharts int
	for songIdStr, chartNotecounts := range notecountsBody {
		songId := 0
		_, err := fmt.Sscanf(songIdStr, "%d", &songId)
		if err != nil {
			log.Println("Error occurred while parsing song ID: ", err)
			context.JSON(500, gin.H{"error": err.Error()})
			return
		}

		for level, notecount := range chartNotecounts {

			// we only care about SP
			if strings.HasPrefix(level, "DP") {
				continue
			}

			// update the chart with the new max score
			rowsAffected, err := gorm.G[models.Chart](db.DB).
				Where("song_id = ? AND level = ?", songId, strings.TrimPrefix(level, "SP")).
				Update(db.DefaultTimeout(), "max_score", notecount*2)

			if err != nil {
				log.Println("Error occurred while updating chart max score: ", err)
				context.JSON(500, gin.H{"error": err.Error()})
				return
			}

			totalUpdatedCharts += int(rowsAffected)
		}
	}

	context.JSON(200, gin.H{
		"message": fmt.Sprintf("Successfully updated max scores for %d charts", totalUpdatedCharts),
	})
}
