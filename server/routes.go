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
	router.GET("/scores", pages.BracketSelect)
	router.GET("/scores/upper", pages.ScoresUpper)
	router.GET("/scores/lower", pages.ScoresLower)
	router.GET("/scores/bracket/:id", pages.BracketSelect)

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
		"danString": func(danLevel int) string {
			return models.DanStrings[danLevel]
		},
		"danColor": func(danLevel int) string {
			if danLevel < 7 {
				return "dan-kyu"
			} else if danLevel <= 14 {
				return "dan-bluedan"
			} else if danLevel <= 16 {
				return "dan-reddan"
			} else if danLevel == 17 {
				return "dan-chuuden"
			} else if danLevel == 18 {
				return "dan-kaiden"
			}
			return ""
		},
		"danStringLatin": func(danLevel int) string {
			return models.DanStringsLatin[danLevel]
		},
	})

	log.Println("Registering custom functions OK")
}
