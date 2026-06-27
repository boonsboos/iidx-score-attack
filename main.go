package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"iidx.boonsboos.nl/server"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/job"
)

func main() {
	//gin.SetMode(gin.ReleaseMode)

	InitLogger()
	config.InitConfig()
	db.InitDB()

	go job.StartWorker()

	InitServer()
}

func InitServer() {
	r := gin.Default()

	server.RegisterRoutes(r)

	r.Use(cachedMiddleware)

	r.LoadHTMLGlob("./client/**/*.html")

	log.Println("Loading OK. Starting server...")

	r.Run(fmt.Sprintf(":%d", config.ServerConfig.Port))
}

func cachedMiddleware(c *gin.Context) {
	c.Writer.Header().Add("Cache-Control", "max-age=86400") // cache everything for one day
	c.Next()
}

func InitLogger() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
