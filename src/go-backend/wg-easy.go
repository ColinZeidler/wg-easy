package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

var RELEASE = 12

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

type clientAddress struct {
	Address string `json:"address"`
}

type clientUri struct {
	ClientId string `uri:"clientId" binding:"required"`
}

type errorMessage struct {
	Error string `json:"error"`
}

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(webConf.Password) == 0 {
			fmt.Println("No password Needed, allow api access")
			ctx.Next()
			return
		}

		session := sessions.Default(ctx)
		value := session.Get("Authenticated")
		authenticated, ok := value.(bool)
		if ok && authenticated {
			ctx.Next()
			return
		}

		if authString, authOk := ctx.Request.Header[http.CanonicalHeaderKey("authorization")]; authOk {
			authString := authString[0]

			auth := strings.SplitN(authString, " ", 2)
			if len(auth) != 2 || auth[0] != "Basic" {
				response := errorMessage{
					Error: "Not Logged In",
				}
				ctx.JSON(http.StatusUnauthorized, response)
				return
			}
			payload, _ := base64.StdEncoding.DecodeString(auth[1])
			test := webConf.User + ":" + webConf.Password
			if test == string(payload) {
				ctx.Next()
				return
			}
		}

		response := errorMessage{
			Error: "Not Logged In",
		}
		ctx.JSON(http.StatusUnauthorized, response)
	}
}

var webConf WebConfig

func main() {
	configPath := flag.String("c", ConfigGetFile(), "Path to the apps config file")
	genConfig := flag.Bool("generate", false, "Generate a default config and exit")
	flag.Parse()
	ConfigSetFile(*configPath)
	webConf = ConfigGetWeb()
	if *genConfig {
		// Exiting now that a config has been generated
		fmt.Println("Created a config at:", *configPath)
		return
	}
	WGsaveConfig()

	router := gin.Default()
	store := cookie.NewStore([]byte("qiouwhjklv")) // TODO gen random secret?
	router.Use(sessions.Sessions("wgeasysession", store))
	router.GET("/api/release", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, RELEASE)
	})
	router.GET("/api/lang", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, webConf.Lang)
	})
	router.GET("/api/ui-traffic-stats", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, webConf.UiTrafficStats)
	})
	router.GET("/api/ui-chart-type", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, webConf.UiChartType)
	})

	router.GET("/api/session", func(ctx *gin.Context) {
		reqPw := len(webConf.Password) > 0
		authenticated := false
		if reqPw {
			session := sessions.Default(ctx)
			value := session.Get("Authenticated")
			var ok bool
			authenticated, ok = value.(bool)
			if !ok {
				authenticated = false
			}
		}
		response := sessionData{
			RequiresPassword: reqPw,
			Authenticated:    authenticated,
		}
		ctx.JSON(http.StatusOK, response)
	})

	router.POST("/api/session", func(ctx *gin.Context) {
		var login loginRequest
		ctx.BindJSON(&login)

		authenticated := login.Password == webConf.Password
		session := sessions.Default(ctx)
		session.Set("Authenticated", authenticated)
		session.Save()

		if authenticated {
			response := successResponse{
				Success: true,
			}
			ctx.JSON(http.StatusOK, response)
		} else {
			response := statusMessage{
				Status:  401,
				Message: "Incorrect password",
			}
			ctx.JSON(http.StatusUnauthorized, response)
		}
	})
	router.DELETE("/api/session", func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		session.Clear()
		session.Save()
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})

	// WireGuard API endpoints
	authGroup := router.Group("/api/wireguard")
	authGroup.Use(AuthMiddleware())
	authGroup.GET("/client", func(ctx *gin.Context) {
		// Get all Clients
		clients := WGgetClients()
		ctx.JSON(http.StatusOK, clients)
	})
	authGroup.GET("/client/:clientId/qrcode.svg", func(ctx *gin.Context) {
		// Get Client config as QR code
		var client clientUri
		ctx.BindUri(&client)
		// TODO need a qrcode api
		qr := WGgetSVG(client.ClientId)
		if qr == "" {
			ctx.String(http.StatusNoContent, "Requested Client does not exist or have a QR code")
		}
		ctx.Header("Content-Type", "image/svg+xml")
		ctx.String(http.StatusOK, qr)
	})
	authGroup.GET("/client/:clientId/configuration", func(ctx *gin.Context) {
		// Get Client config
		var client clientUri
		ctx.BindUri(&client)

		config := WGgetClientConfig(client.ClientId)
		ctx.Header("Content-Disposition", "attachment; filename="+client.ClientId+".conf")
		ctx.Header("Content-Type", "text/plain")
		ctx.String(http.StatusOK, config)
	})
	authGroup.POST("/client", func(ctx *gin.Context) {
		// Create Client
		var client clientName
		ctx.BindJSON(&client)
		ok := WGcreateClient(client.Name)
		if !ok {
			response := errorMessage{
				Error: "No available IPs",
			}
			ctx.JSON(http.StatusBadRequest, response)
		}
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})
	authGroup.DELETE("/client/:clientId", func(ctx *gin.Context) {
		// Delete Client
		var client clientUri
		ctx.BindUri(&client)

		WGdeleteClient(client.ClientId)
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})
	authGroup.POST("/client/:clientId/enable", func(ctx *gin.Context) {
		// Enable Client
		var client clientUri
		ctx.BindUri(&client)

		WGenableClient(client.ClientId)
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})
	authGroup.POST("/client/:clientId/disable", func(ctx *gin.Context) {
		// Disable Client
		var client clientUri
		ctx.BindUri(&client)

		WGdisableClient(client.ClientId)
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})
	authGroup.PUT("/client/:clientId/name/", func(ctx *gin.Context) {
		// Update Client name
		var client clientUri
		ctx.BindUri(&client)
		var cName clientName
		err := ctx.BindJSON(&cName)

		if err != nil {
			return
		}
		fmt.Println(cName, err)

		WGupdateClientName(client.ClientId, cName.Name)
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})
	authGroup.PUT("/client/:clientId/address/", func(ctx *gin.Context) {
		// Update Client address
		var client clientUri
		ctx.BindUri(&client)
		var cAddress clientAddress
		err := ctx.BindJSON(&cAddress)
		if err != nil {
			return
		}

		WGupdateClientAddress(client.ClientId, cAddress.Address)
		response := successResponse{
			Success: true,
		}
		ctx.JSON(http.StatusOK, response)
	})

	router.Run(":" + webConf.Port)
}
