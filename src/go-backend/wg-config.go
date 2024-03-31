package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
)

type AppConfig struct {
	WGApiConfig `json:"wgConfig"`
	WebConfig   `json:"webConfig"`
}

type WGApiConfig struct {
	Path            string
	HostName        string
	Port            string
	Mtu             string
	DefaultDNS      string
	DefaultAddress  string
	AllowedServerIp string
	PreUp           string
	PostUp          string
	PreDown         string
	PostDown        string
	Interface       string
}

type WebConfig struct {
	Lang           string
	UiTrafficStats bool
	UiChartType    int
	Password       string
	Port           string
	CookieSecret   string
	User           string
}

var localAppConfig *AppConfig = nil
var configFile string = "./wg-easy.json"

func ConfigSetFile(filePath string) {
	configFile = filePath
}

func ConfigGetAppConf() *AppConfig {
	if localAppConfig == nil {
		jsonBytes, err := os.ReadFile(configFile)

		// generate a Inital secret to use if not set in Config
		idBytes := make([]byte, 10)
		rand.Read(idBytes)
		cookieSecret := hex.EncodeToString(idBytes)
		// Configure default conf
		conf := AppConfig{
			WGApiConfig: WGApiConfig{
				Path:            "/etc/wireguard",
				Port:            "51820",
				Interface:       "wg0",
				AllowedServerIp: "0.0.0.0/0, ::/0",
				DefaultAddress:  "10.0.0.x",
			},
			WebConfig: WebConfig{
				Lang:           "en",
				UiTrafficStats: true,
				UiChartType:    0,
				Port:           "9505",
				Password:       "REPLACE_ME",
				CookieSecret:   cookieSecret,
			},
		}
		if err != nil {
			ConfigSaveAppConf(&conf)
			return &conf
		}
		jsonErr := json.Unmarshal(jsonBytes, &conf)
		if jsonErr != nil {
			ConfigSaveAppConf(&conf)
			return &conf
		}
		localAppConfig = &conf
	}

	return localAppConfig
}

func ConfigSaveAppConf(conf *AppConfig) {
	jsonBytes, _ := json.MarshalIndent(conf, "", "\t")
	os.WriteFile(configFile, jsonBytes, 0o600)
}

func ConfigGetApi() WGApiConfig {
	appConfig := ConfigGetAppConf()
	return appConfig.WGApiConfig
}

func ConfigGetWeb() WebConfig {
	appConfig := ConfigGetAppConf()
	return appConfig.WebConfig
}
