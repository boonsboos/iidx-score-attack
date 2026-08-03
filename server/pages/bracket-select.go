package pages

import (
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"iidx.boonsboos.nl/server/db"
	"iidx.boonsboos.nl/server/models"
)

func BracketSelect(context *gin.Context) {
	pageNumberS := context.Query("page")
	if pageNumberS == "" {
		pageNumberS = "1"
	}

	pageNumber, err := strconv.Atoi(pageNumberS)
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}

	brackets := lo.Map(db.GetBracketsPaginated(25, (pageNumber-1)*25), func(bracket models.BracketListBracketTypes, i int) models.FrontendBracketListBracket {
		return models.FrontendBracketListBracket{
			Title:        bracket.Title,
			ActiveFrom:   bracket.ActiveFrom.Format("2006-01-02"),
			ActiveUntil:  bracket.ActiveUntil.Format("2006-01-02"),
			BracketTypes: bracket.BracketTypes,
		}
	})

	sort.SliceStable(brackets, func(i, j int) bool {
		return brackets[i].ActiveFrom > brackets[j].ActiveFrom
	})

	context.HTML(200, "bracket-select.html", gin.H{
		"Brackets": brackets,
		"Page":     pageNumber,
	})
}
