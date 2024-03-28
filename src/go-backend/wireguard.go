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

func WGgetConfig() WGConfig {
	jsonBytes, err := os.ReadFile(filepath.Join(WG_PATH, "wg0.json"))
	var wgConfig WGConfig
	if err == nil {
		jsonErr := json.Unmarshal(jsonBytes, &wgConfig)
		if jsonErr != nil {
			wgConfig = WGcreateNewConfig()
		}
	} else {
		wgConfig = WGcreateNewConfig()
	}

	fmt.Println(wgConfig)
	_WGsaveConfig(wgConfig)
	return wgConfig
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

func _WGsaveConfig(wgConfig WGConfig) {
	jsonBytes, err := json.Marshal(wgConfig)
	if err != nil {
		fmt.Println("Issue Encoding config to JSON", err)
	}
	err = os.WriteFile(filepath.Join(WG_PATH, "wg0.json"), jsonBytes, 0o600)
	if err != nil {
		fmt.Println("Issue writing config", err)
	}

	// TODO save wg0.conf file
}

func _WGsyncConfig() {
	cmd := exec.Command("wg", "syncconfg", "wg0", filepath.Join(WG_PATH, "wg0.conf"))
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error syncing wg config: ", err)
	}

}

func WGgetStats() (string, bool) {
	cmd := exec.Command("wg", "show", "wg0", "dump")
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
	configBuilder.WriteString("\nAddress = ")
	configBuilder.WriteString(client.Address)
	configBuilder.WriteString("/24")
	if WG_DEFAULT_DNS != "" {
		configBuilder.WriteString("\nDNS = ")
		configBuilder.WriteString(WG_DEFAULT_DNS)
	}
	if WG_MTU != "" {
		configBuilder.WriteString("\nMTU = ")
		configBuilder.WriteString(WG_MTU)
	}

	configBuilder.WriteString("\n\n[Peer]\nPublicKey = ")
	configBuilder.WriteString(config.Server.PublicKey)
	configBuilder.WriteString("\nAllowedIPs = ")
	configBuilder.WriteString(WG_ALLOWED_IPS)
	configBuilder.WriteString("\nEndpoint = ")
	configBuilder.WriteString(WG_HOST)
	configBuilder.WriteString(":")
	configBuilder.WriteString(WG_PORT)

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
	cmd := exec.Command("wg-quick", "down", "wg0")
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error bringing wg0 down: ", err)
	}
}
