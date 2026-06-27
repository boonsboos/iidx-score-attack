package pages

import "github.com/gin-gonic/gin"

func Success(context *gin.Context) {
	djName := context.Query("dj_name")
	if len(djName) == 0 {
		context.Redirect(302, "/")
		return
	}

	context.HTML(200, "success.html", gin.H{
		"DJName": djName,
	})
}
