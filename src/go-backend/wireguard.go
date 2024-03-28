package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var WG_PATH = "./"
var WG_HOST = ""
var WG_PORT = "51820"
var WG_MTU = ""
var WG_DEFAULT_DNS = ""
var WG_DEFAULT_ADDRESS = "10.0.0.x"
var WG_PERSISTENT_KEEPALIVE = ""
var WG_ALLOWED_IPS = "0.0.0.0/0, ::/0"
var WG_PRE_UP = ""
var WG_POST_UP = ""
var WG_PRE_DOWN = ""
var WG_POST_DOWN = ""
var WG_INTERFACE = "wg0"

var TESTING = false

type WGConfig struct {
	Server  WGServer            `json:"server"`
	Clients map[string]WGClient `json:"clients"`
}

type WGServer struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
}

type WGClient struct {
	ClientId            string `json:"id"`
	Name                string `json:"name"`
	Enabled             string `json:"enabled"`
	Address             string `json:"address"`
	PublicKey           string `json:"publicKey"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
	AllowedIPs          string `json:"allowedIPs"`
	DownloadableConfig  bool   `json:"downloadableConfig"`
	PersistentKeepalive string `json:"persistentKeepAlive"`
	TransferRx          int    `json:"transferRx"`
	TransferTx          int    `json:"transferTx"`
	LatestHandshakeAt   string `json:"latestHandshakeAt"`
	PrivateKey          string `json:"privateKey,omitempty"`
}

var myConfig *WGConfig = nil

func WGgetConfig() *WGConfig {
	if myConfig == nil {
		jsonBytes, err := os.ReadFile(filepath.Join(WG_PATH, WG_INTERFACE+".json"))
		var wgConfig WGConfig
		if err == nil {
			jsonErr := json.Unmarshal(jsonBytes, &wgConfig)
			if jsonErr != nil {
				wgConfig = WGcreateNewConfig()
			}
		} else {
			wgConfig = WGcreateNewConfig()
		}
		myConfig = &wgConfig
		_WGsaveConfig(myConfig)
		// TODO first time startup of config with wg-quick down up
		// _WGsyncConfig()
	}
	return myConfig
}

func WGcreateNewConfig() WGConfig {
	outBytes, err := exec.Command("wg", "genkey").Output()
	if err != nil {
		fmt.Println("Error creating Private Key")
		panic(err)
	}
	privateKey := string(outBytes)
	privateKey = strings.Trim(privateKey, "\n")

	cmdPipe := "echo " + privateKey + " | wg pubkey"
	pubBytes, pubErr := exec.Command("bash", "-c", cmdPipe).Output()
	if pubErr != nil {
		fmt.Println("Error creating Public Key")
		panic(pubErr)
	}
	publicKey := string(pubBytes)
	publicKey = strings.Trim(publicKey, "\n")

	address := strings.Replace(WG_DEFAULT_ADDRESS, "x", "1", -1)

	config := WGConfig{
		Server: WGServer{
			PrivateKey: privateKey,
			PublicKey:  publicKey,
			Address:    address,
		},
	}
	return config
}

func WGsaveConfig() {
	config := WGgetConfig()
	_WGsaveConfig(config)
	_WGsyncConfig()
}

func _WGsaveConfig(wgConfig *WGConfig) {
	jsonBytes, err := json.MarshalIndent(wgConfig, "", "\t")
	if err != nil {
		fmt.Println("Issue Encoding config to JSON", err)
	}
	err = os.WriteFile(filepath.Join(WG_PATH, WG_INTERFACE+".json"), jsonBytes, 0o600)
	if err != nil {
		fmt.Println("Issue writing config", err)
	}

	// save WG_INTERFACE.conf file
	var configBuilder strings.Builder

	configBuilder.WriteString("# Server\n")
	configBuilder.WriteString("[Interface]\n")
	configBuilder.WriteString("PrivateKey = " + wgConfig.Server.PrivateKey + "\n")
	configBuilder.WriteString("Address = " + wgConfig.Server.Address + "/24\n")
	configBuilder.WriteString("ListenPort = " + WG_PORT + "\n")
	configBuilder.WriteString("PreUp = " + WG_PRE_UP + "\n")
	configBuilder.WriteString("PostUp = " + WG_POST_UP + "\n")
	configBuilder.WriteString("PreDown = " + WG_PRE_DOWN + "\n")
	configBuilder.WriteString("PostDown = " + WG_POST_DOWN + "\n")
	configBuilder.WriteString("\n")

	for clientId, client := range wgConfig.Clients {
		configBuilder.WriteString("# Client" + client.Name + " (" + clientId + ")\n")
		configBuilder.WriteString("[Peer]\n")
		configBuilder.WriteString("PublicKey = " + client.PublicKey + "\n")
		configBuilder.WriteString("PublicKey = " + client.PublicKey + "\n")
		configBuilder.WriteString("AllowedIPs = " + client.AllowedIPs + "/32\n")
	}

	configString := configBuilder.String()

	confErr := os.WriteFile(filepath.Join(WG_PATH, WG_INTERFACE+".conf"), []byte(configString), 0o600)
	if confErr != nil {
		fmt.Println("Issue writing .conf config", err)
	}

}

func _WGsyncConfig() {
	if TESTING {
		fmt.Println("Skipping due to testing")
		return
	}
	pipeCmd := "wg syncconf " + WG_INTERFACE + " <(wg-quick strip " + WG_INTERFACE + ")"
	cmd := exec.Command("bash", "-c", pipeCmd)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error syncing wg config: ", err)
	}
}

func WGgetStats() (string, bool) {
	cmd := exec.Command("wg", "show", WG_INTERFACE, "dump")
	var statsB bytes.Buffer
	cmd.Stdout = &statsB
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error getting WG stats: ", err)
		return "", false
	}
	return statsB.String(), true
}

func WGgetClients() []WGClient {
	config := WGgetConfig()

	clients := config.Clients

	stats, ok := WGgetStats()
	if ok { //if there was an issue gettings stats dont add them
		for index, line := range strings.Split(stats, "\n") {
			// First line doesn't match all the peer lines, so skip it
			if index == 0 {
				continue
			}
			//Data is: pubKey 0, PreSK 1, endpoint 2, allowedIps 3, latestHS 4, RX 5, TX 6, persistKA 7
			data := strings.Split(line, "\t")
			publicKey := data[0]
			lastHandshake := data[4]
			rx, err := strconv.Atoi(data[5])
			if err != nil {
				rx = 0
			}
			tx, err := strconv.Atoi(data[6])
			if err != nil {
				tx = 0
			}
			persist := data[7]
			client := clients[publicKey]
			client.LatestHandshakeAt = lastHandshake // TODO parse int and convert to date string like "2024-03-26T21:56:41.430Z"
			client.TransferRx = rx
			client.TransferTx = tx
			client.PersistentKeepalive = persist

			// TODO test if this needs to be done:
			clients[publicKey] = client
		}
	}

	clientArr := make([]WGClient, 0, len(clients))
	for _, c := range clients {
		clientArr = append(clientArr, c)
	}

	return clientArr
}

/*
* Returns Zero value  WGClient if no such client exists
 */
func WGgetClient(clientId string) (WGClient, bool) {
	config := WGgetConfig()
	client, ok := config.Clients[clientId]
	return client, ok
}

func WGgetClientConfig(clientId string) (string, bool) {
	config := WGgetConfig()
	client, ok := WGgetClient(clientId)
	if !ok {
		return "", false
	}

	var configBuilder strings.Builder
	configBuilder.WriteString("[Interface]\nPrivateKey = ")
	if client.PrivateKey != "" {
		configBuilder.WriteString("REPLACE_ME")
	} else {
		configBuilder.WriteString(client.PrivateKey)
	}
	configBuilder.WriteString("\nAddress = " + client.Address + "/24")
	if WG_DEFAULT_DNS != "" {
		configBuilder.WriteString("\nDNS = " + WG_DEFAULT_DNS)
	}
	if WG_MTU != "" {
		configBuilder.WriteString("\nMTU = " + WG_MTU)
	}

	configBuilder.WriteString("\n\n[Peer]\nPublicKey = " + config.Server.PublicKey)
	configBuilder.WriteString("\nAllowedIPs = " + WG_ALLOWED_IPS)
	configBuilder.WriteString("\nEndpoint = " + WG_HOST + ":" + WG_PORT)
	configBuilder.WriteString("\n")

	clientConfig := configBuilder.String()
	return clientConfig, true
}

func WGcreateClient(clientId string) {

}

func WGdeleteClient(clientId string) {

}

func WGenableClient(clientId string) {

}

func WGdisableClient(clientId string) {

}

func WGupdateClientName(clientId string, name string) {

}

func WGupdateClientAddress(clientId string, address string) {

}

func WGshutdown() {
	if TESTING {
		fmt.Println("WGshutdown, Skipping due to Testing")
		return
	}
	cmd := exec.Command("wg-quick", "down", WG_INTERFACE)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error bringing", WG_INTERFACE, "down:", err)
	}
}
