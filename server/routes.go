package server

import (
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

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

	router.GET("/scores/:startDate/:bracket", pages.ScoresGeneric)

	router.GET("/scores/bracket/:id", pages.BracketSelect)
	router.GET("/privacy-policy", pages.CookiePrivacy)

	// alias until deployments are updated
	router.POST("/oauth/callback", thirdparty.HandleFOauthCallback)
	router.GET("/oauth/callback", thirdparty.HandleFOauthCallback)

	router.POST("/oauth/callback-f", thirdparty.HandleFOauthCallback)
	router.GET("/oauth/callback-f", thirdparty.HandleFOauthCallback)
	router.POST("/oauth/callback-k", thirdparty.HandleKOauthCallback)
	router.GET("/oauth/callback-k", thirdparty.HandleKOauthCallback)

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
		"danStringLatin": func(danLevel int) string {
			return models.DanStringsLatin[danLevel]
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
		"bracketColor": func(bracketType string) string {
			switch bracketType {
			case "ANOTHER":
				return "btn-danger"
			case "master":
				return "btn-danger"
			case "HYPER":
				return "btn-warning"
			case "NORMAL":
				return "btn-primary"
			case "upper":
				return "btn-primary"
			case "BEGINNER":
				return "btn-success"
			case "lower":
				return "btn-success"
			default:
				return ""
			}
		},
		"capitalize": func(s string) string {
			return strings.ToTitle(s)
		},
		"join": func(arr []string, sep string) string {
			return strings.Join(arr, sep)
		},
		"unix": func(s string) int64 {
			t, _ := time.Parse("2006-01-02", s)
			return t.Unix()
		},
	})

	log.Println("Registering custom functions OK")
}
