package db

import (
	"fmt"
	"log"
	"sort"
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

func GetChartPool(startTime time.Time) (models.ChartPool, error) {
	pool, err := gorm.G[models.ChartPool](DB).
		Where("active_from = ?", startTime).
		First(DefaultTimeout())
	if err != nil {
		return models.ChartPool{}, err
	}
	return pool, nil
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

func GetCurrentlyActiveChartPoolStartTime() (time.Time, time.Time, error) {
	activePool, err := GetCurrentlyActiveChartPool()
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return activePool.ActiveFrom, activePool.ActiveUntil, nil
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

func GetBracketsPaginated(limit int, offset int) []models.BracketListBracketTypes {
	var brackets []models.BracketChart

	// fetch the latest brackets for each bracket type
	brackets, err := gorm.G[models.BracketChart](DB).
		Joins(clause.Has("Pool"), nil).
		Group("Pool.title, bracket_type").
		Having("Pool.active_from <= ?", time.Now()).
		Order("Pool.active_from DESC").
		Find(DefaultTimeout())
	if err != nil {
		log.Println("Error fetching brackets: ", err)
		return []models.BracketListBracketTypes{}
	}

	raw := lo.Map(brackets, func(bracket models.BracketChart, i int) models.BracketListBracket {
		return models.BracketListBracket{
			Title:       bracket.Pool.Title,
			ActiveFrom:  bracket.Pool.ActiveFrom,
			ActiveUntil: bracket.Pool.ActiveUntil,
			BracketType: bracket.BracketType,
		}
	})

	grouped := lo.GroupBy(raw, func(item models.BracketListBracket) string {
		return item.Title
	})

	return lo.MapToSlice(grouped, func(title string, items []models.BracketListBracket) models.BracketListBracketTypes {
		// merge the bracket types together
		bracketTypes := lo.Map(items, func(item models.BracketListBracket, in int) string {
			return item.BracketType
		})

		sortBracketTypes(bracketTypes)

		return models.BracketListBracketTypes{
			Title:        title,
			ActiveFrom:   items[0].ActiveFrom,
			ActiveUntil:  items[0].ActiveUntil,
			BracketTypes: bracketTypes,
		}
	})
}

var order map[string]int = map[string]int{
	"lower":    1,
	"upper":    2,
	"master":   3,
	"beginner": 1,
	"normal":   2,
	"hyper":    3,
	"another":  4,
}

// sort array in place in the correct order: lower, upper, master OR beginner, normal, hyper, another
func sortBracketTypes(bracketTypes []string) {
	sort.SliceStable(bracketTypes, func(i, j int) bool {

		return order[bracketTypes[i]] < order[bracketTypes[j]]
	})
}
