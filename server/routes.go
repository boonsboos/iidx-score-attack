package server

import (
	"fmt"
	"log"
	"text/template"

	"github.com/gin-gonic/gin"
	"iidx.boonsboos.nl/server/models"
	"iidx.boonsboos.nl/server/pages"
	"iidx.boonsboos.nl/server/thirdparty"
)

func RegisterRoutes(router *gin.Engine) {

	log.Println("Registering routes...")

	router.Static("/static", "./client/static/")

	router.GET("/", pages.Index)
	router.GET("/success", pages.Success)
	router.GET("/scores", pages.Scores)

	router.POST("/oauth/callback", thirdparty.HandleOauthCallback)
	router.GET("/oauth/callback", thirdparty.HandleOauthCallback)

	RegisterMaintenanceRoutes(router)

	log.Println("Registering routes OK")

	log.Println("Registering custom functions...")

	router.SetFuncMap(template.FuncMap{
		"lampString": func(lamp int) string {
			return models.LampStrings[lamp]
		},
		"ratingFormat": func(rating float64) string {
			return fmt.Sprintf("%.2f", rating)
		},
	})

	log.Println("Registering custom functions OK")
}
