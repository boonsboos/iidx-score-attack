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
		upper, lower, err := db.GetPoolChartsFrontend(activeChartPool)
		if err != nil {
			log.Println("Error fetching active charts for frontend:", err)
		} else {
			upperBracketCharts = upper
			lowerBracketCharts = lower
		}
	}

	context.HTML(http.StatusOK, "index.html", gin.H{
		"BracketActive":       activeChartPool.ID != 0,
		"ClientId":            config.ServerConfig.OauthClientId,
		"RedirectURI":         config.ServerConfig.OauthRedirectUrl,
		"PoolName":            activeChartPool.Title,
		"StartTime":           activeChartPool.ActiveFrom.Format("02-01-2006"),  // for the bracket countdown timer
		"EndTime":             activeChartPool.ActiveUntil.Format("02-01-2006"), // for the bracket countdown timer
		"MasterBracketCharts": masterBracketCharts,
		"UpperBracketCharts":  upperBracketCharts,
		"LowerBracketCharts":  lowerBracketCharts,
	})
}
