package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	ginsession "github.com/go-session/gin-session"
)

var RELEASE = 1
var LANG = "en"
var UI_TRAFFIC_STATS = false
var UI_CHART_TYPE = 0
var PASSWORD = ""

type sessionData struct {
	RequiresPassword bool `json:"requiresPassword"`
	Authenticated    bool `json:"authenticated"`
}

type successResponse struct {
	Success bool `json:"success"`
}

type loginRequest struct {
	Password string `json:"password"`
}

type statusMessage struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type clientName struct {
	Name string `json:"name"`
}

type clientUri struct {
	ClientId string `uri:"clientId" binding:"required"`
}

type errorMessage struct {
	Error string `json:"error"`
}

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fmt.Println("Checking Auth Middleware")
		if len(PASSWORD) == 0 {
			fmt.Println("No password Needed, allow api access")
			ctx.Next()
			return
		}

		session := ginsession.FromContext(ctx)
		value, ok := session.Get("Authenticated")
		if ok {
			authenticated, ok := value.(bool)
			if ok && authenticated {
				fmt.Println("Authentication Success")
				ctx.Next()
				return
			}
		}

		response := errorMessage{
			Error: "Not Logged In",
		}
		ctx.IndentedJSON(http.StatusUnauthorized, response)
	}
}

func main() {
	router := gin.Default()
	router.Use(ginsession.New())
	router.GET("/api/release", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, RELEASE)
	})
	router.GET("/api/lang", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, LANG)
	})
	router.GET("/api/ui-traffic-stats", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, UI_TRAFFIC_STATS)
	})
	router.GET("/api/ui-chart-type", func(ctx *gin.Context) {
		ctx.IndentedJSON(http.StatusOK, UI_CHART_TYPE)
	})

	router.GET("/api/session", func(ctx *gin.Context) {
		reqPw := len(PASSWORD) > 0
		authenticated := false
		if reqPw {
			session := ginsession.FromContext(ctx)
			value, ok := session.Get("Authenticated")
			if !ok {
				authenticated = false
			}
			authenticated, ok = value.(bool)
			if !ok {
				authenticated = false
			}
		}
		response := sessionData{
			RequiresPassword: reqPw,
			Authenticated:    authenticated,
		}
		ctx.IndentedJSON(http.StatusOK, response)
	})

	router.POST("/api/session", func(ctx *gin.Context) {
		var login loginRequest
		ctx.BindJSON(&login)

		authenticated := login.Password == PASSWORD
		session := ginsession.FromContext(ctx)
		session.Set("Authenticated", authenticated)

		if authenticated {
			response := successResponse{
				Success: true,
			}
			ctx.IndentedJSON(http.StatusOK, response)
		} else {
			response := statusMessage{
				Status:  401,
				Message: "Incorrect password",
			}
			ctx.IndentedJSON(http.StatusUnauthorized, response)
		}
	})

	// WireGuard API endpoints
	authGroup := router.Group("/api/wireguard")
	authGroup.Use(AuthMiddleware())
	authGroup.DELETE("/api/session", func(ctx *gin.Context) {
		ginsession.Destroy(ctx)
		response := successResponse{
			Success: true,
		}
		ctx.IndentedJSON(http.StatusOK, response)
	})
	authGroup.GET("/client", func(ctx *gin.Context) {
		// Get all Clients
		fmt.Println("Get Clients")
		WGgetClients()
	})
	authGroup.GET("/client/:clientId/qrcode.svg", func(ctx *gin.Context) {
		// Get Client config as QR code
		var client clientUri
		ctx.BindUri(&client)
		// TODO need a qrcode api
	})
	authGroup.GET("/client/:clientId/configuration", func(ctx *gin.Context) {
		// Get Client config
		var client clientUri
		ctx.BindUri(&client)

	})
	authGroup.POST("/client", func(ctx *gin.Context) {
		// Create Client
		var client clientName
		ctx.BindJSON(&client)
		WGcreateClient(client.Name)
		response := successResponse{
			Success: true,
		}
		ctx.IndentedJSON(http.StatusOK, response)
	})
	authGroup.DELETE("/client/:clientId", func(ctx *gin.Context) {
		// Delete Client
		var client clientUri
		ctx.BindUri(&client)

		WGdeleteClient(client.ClientId)

	})
	authGroup.POST("/client/:clientId/enable", func(ctx *gin.Context) {
		// Enable Client
	})
	authGroup.POST("/client/:clientId/disable", func(ctx *gin.Context) {
		// Disable Client
	})
	authGroup.PUT("/client/:clientId/name", func(ctx *gin.Context) {
		// Update Client name
	})
	authGroup.PUT("/client/:clientId/address", func(ctx *gin.Context) {
		// Update Client address
	})

	router.Run(":9505")
}
