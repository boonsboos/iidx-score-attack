package pages

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

func Index(context *gin.Context) {

	upperBracketCharts := make([]models.FrontendBracketChart, 0)
	lowerBracketCharts := make([]models.FrontendBracketChart, 0)

	activeChartPool, err := db.GetCurrentlyActiveChartPool()
	if err == nil {
		upper, lower, err := db.GetPoolChartsFrontend(activeChartPool)
		if err == nil {
			upperBracketCharts = upper
			lowerBracketCharts = lower
		}
	}

	context.HTML(http.StatusOK, "index.html", gin.H{
		"ClientId":           config.ServerConfig.OauthClientId,
		"RedirectURI":        config.ServerConfig.OauthRedirectUrl,
		"PoolName":           activeChartPool.Title,
		"StartTime":          activeChartPool.ActiveFrom.Format("02-01-2006"),  // for the bracket countdown timer
		"EndTime":            activeChartPool.ActiveUntil.Format("02-01-2006"), // for the bracket countdown timer
		"UpperBracketCharts": upperBracketCharts,
		"LowerBracketCharts": lowerBracketCharts,
	})
}
