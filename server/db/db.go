package db

import (
	"context"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"iidx.boonsboos.nl/server/config"
	"iidx.boonsboos.nl/server/models"
)

var DB *gorm.DB

func InitDB() {
	log.Println("Loading DB...")

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       config.ServerConfig.ConnectionString + "?parseTime=true",
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic("failed to connect database")
	}

	log.Println("Migrating DB...")
	db.AutoMigrate(&models.ChartPool{}, &models.Score{}, &models.BracketChart{}, &models.Chart{}, &models.GameVersion{})

	sqlDb, _ := db.DB()

	sqlDb.SetMaxIdleConns(10)
	sqlDb.SetMaxOpenConns(100)
	sqlDb.SetConnMaxLifetime(time.Hour)

	go func() {
		for {
			time.Sleep(time.Second * 15)
			err := sqlDb.Ping()
			if err != nil {
				log.Println("Healthcheck | DB ping failed:", err)
			}
		}
	}()

	DB = db
	log.Println("Migrating DB OK")

	log.Println("Loading DB OK")
}

// returns a default timeout of 30 seoncds for use with gorm operations, to prevent hanging connections
func DefaultTimeout() context.Context {
	a, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return a
}
