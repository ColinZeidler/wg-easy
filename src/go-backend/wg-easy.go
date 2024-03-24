package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var release = 1
var lang = "en"
var ui_traffic_stats = false
var ui_chart_type = 0

func main() {
	router := gin.Default()
	router.GET("/api/release", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, release)
	})
	router.GET("/api/lang", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, lang)
	})
	router.GET("/api/ui-traffic-stats", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, ui_traffic_stats)
	})
	router.GET("/api/ui-chart-type", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, ui_chart_type)
	})

	router.GET("/api/session", func(ctx *gin.Context) {
		// returns { bool, bool } list requiresPassword, authenticated
	})

	router.POST("/api/session", func(ctx *gin.Context) {

	})

	router.Run(":9505")
}
