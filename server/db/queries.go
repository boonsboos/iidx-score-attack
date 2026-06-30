package db

import (
	"fmt"
	"strconv"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"iidx.boonsboos.nl/server/models"
)

func GetCurrentlyActiveChartPool() (models.ChartPool, error) {
	return gorm.G[models.ChartPool](DB).
		Where("active_from <= ? AND active_until >= ?", time.Now(), time.Now()).
		First(DefaultTimeout())
}

func GetPoolCharts(pool models.ChartPool) ([]models.BracketChart, error) {
	return gorm.G[models.BracketChart](DB).
		Joins(clause.JoinTarget{Association: "Chart"}, nil).
		Where("pool_id = ?", pool.ID).
		Find(DefaultTimeout())
}

func GetPoolChartsScoresPage(pool models.ChartPool) ([]models.BracketChart, error) {

	var allCharts []models.BracketChart

	err := DB.Model(&models.BracketChart{}).
		Joins("Chart").
		Joins("Chart.Song").
		Where("pool_id = ?", pool.ID).
		Order("bracket_charts.chart_type DESC, Chart__Song__name ASC").
		Scan(&allCharts).Error

	if err != nil {
		return nil, fmt.Errorf("error occurred while fetching pool charts: %w", err)
	}

	return allCharts, nil
}

func GetPoolChartsFrontend(pool models.ChartPool) ([]models.FrontendBracketChart, []models.FrontendBracketChart, []models.FrontendBracketChart, error) {
	var allCharts []models.BracketChart

	err := DB.Model(&models.BracketChart{}).
		Joins("Chart").
		Joins("Chart.Song").
		Joins("Chart.Song.Version").
		Where("pool_id = ?", pool.ID).
		Order("bracket_charts.chart_type DESC, Chart__Song__name ASC").
		Scan(&allCharts).Error
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error occurred while fetching pool charts: %w", err)
	}

	masterCharts := filterByBracketType(allCharts, "master")
	upperCharts := filterByBracketType(allCharts, "upper")
	lowerCharts := filterByBracketType(allCharts, "lower")

	masterFrontendCharts := mapToFrontendBracketChart(masterCharts)
	upperFrontendCharts := mapToFrontendBracketChart(upperCharts)
	lowerFrontendCharts := mapToFrontendBracketChart(lowerCharts)

	return masterFrontendCharts, upperFrontendCharts, lowerFrontendCharts, nil
}

func filterByBracketType(charts []models.BracketChart, bracketType string) []models.BracketChart {
	return lo.Filter(charts, func(chart models.BracketChart, i int) bool {
		return chart.BracketType == bracketType
	})
}

func mapToFrontendBracketChart(charts []models.BracketChart) []models.FrontendBracketChart {
	return lo.Map(charts, func(chart models.BracketChart, i int) models.FrontendBracketChart {
		return models.FrontendBracketChart{
			Title:          chart.Chart.Song.Name,
			TitleLatinized: chart.Chart.Song.NameLatinized,
			Artist:         chart.Chart.Song.Artist,
			ChartLevel:     "SP" + chart.Chart.Difficulty + strconv.Itoa(chart.Chart.Level),
			Version:        chart.Chart.Song.Version.Name,
			VersionId:      chart.Chart.Song.Version.ID,
			ChartType:      chart.ChartType,
		}
	})
}
