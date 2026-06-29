package pages

import (
	"github.com/gin-gonic/gin"
)

func CookiePrivacy(context *gin.Context) {
	context.HTML(200, "cookie-privacy.html", gin.H{})
}
