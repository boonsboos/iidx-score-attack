package pages

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

func Index(context *gin.Context) {
	var beginnerBracketCharts models.FrontendBracket
	var normalBracketCharts models.FrontendBracket
	var hyperBracketCharts models.FrontendBracket
	var anotherBracketCharts models.FrontendBracket

	activeChartPool, err := db.GetCurrentlyActiveChartPool()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("No active chart pool found")
		} else {
			log.Println("Error fetching active chart pool:", err)
		}
	} else {
		charts, err := db.GetPoolChartsFrontend(activeChartPool)
		if err != nil {
			log.Println("Error fetching active charts for frontend:", err)
		} else {
			beginnerBracketCharts = charts["beginner"]
			normalBracketCharts = charts["normal"]
			hyperBracketCharts = charts["hyper"]
			anotherBracketCharts = charts["another"]
		}
	}

	context.HTML(http.StatusOK, "index.html", gin.H{
		"FAuthURI":     config.ServerConfig.GetApiConfigByName("F").AuthBaseUrl,
		"FClientId":    config.ServerConfig.GetApiConfigByName("F").ClientId,
		"FRedirectURI": config.ServerConfig.GetApiConfigByName("F").OauthRedirectUrl,
		"KAuthURI":     config.ServerConfig.GetApiConfigByName("K").AuthBaseUrl,
		"KClientId":    config.ServerConfig.GetApiConfigByName("K").ClientId,

		"BracketActive":         activeChartPool.ID != 0,
		"PoolName":              activeChartPool.Title,
		"StartTime":             activeChartPool.ActiveFrom.Format("2006-01-02"),  // for the bracket countdown timer
		"EndTime":               activeChartPool.ActiveUntil.Format("2006-01-02"), // for the bracket countdown timer
		"BeginnerBracketCharts": beginnerBracketCharts,
		"NormalBracketCharts":   normalBracketCharts,
		"HyperBracketCharts":    hyperBracketCharts,
		"AnotherBracketCharts":  anotherBracketCharts,
	})
}
