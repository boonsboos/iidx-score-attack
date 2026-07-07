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
	masterBracketCharts := make([]models.FrontendBracketChart, 0)
	upperBracketCharts := make([]models.FrontendBracketChart, 0)
	lowerBracketCharts := make([]models.FrontendBracketChart, 0)

	activeChartPool, err := db.GetCurrentlyActiveChartPool()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("No active chart pool found")
		} else {
			log.Println("Error fetching active chart pool:", err)
		}
	} else {
		master, upper, lower, err := db.GetPoolChartsFrontend(activeChartPool)
		if err != nil {
			log.Println("Error fetching active charts for frontend:", err)
		} else {
			masterBracketCharts = master
			upperBracketCharts = upper
			lowerBracketCharts = lower
		}
	}

	context.HTML(http.StatusOK, "index.html", gin.H{
		"FAuthURI":     config.ServerConfig.GetApiConfigByName("F").AuthBaseUrl,
		"FClientId":    config.ServerConfig.GetApiConfigByName("F").ClientId,
		"FRedirectURI": config.ServerConfig.GetApiConfigByName("F").OauthRedirectUrl,
		"KAuthURI":     config.ServerConfig.GetApiConfigByName("K").AuthBaseUrl,
		"KClientId":    config.ServerConfig.GetApiConfigByName("K").ClientId,

		"BracketActive":       activeChartPool.ID != 0,
		"PoolName":            activeChartPool.Title,
		"StartTime":           activeChartPool.ActiveFrom.Format("02-01-2006"),  // for the bracket countdown timer
		"EndTime":             activeChartPool.ActiveUntil.Format("02-01-2006"), // for the bracket countdown timer
		"MasterBracketCharts": masterBracketCharts,
		"UpperBracketCharts":  upperBracketCharts,
		"LowerBracketCharts":  lowerBracketCharts,
	})
}
