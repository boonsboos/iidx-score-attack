package pages

import "github.com/gin-gonic/gin"

func BracketSelect(context *gin.Context) {
	context.HTML(200, "bracket-select.html", nil)
}
