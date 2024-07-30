package main

import (
	"github.com/livekit/protocol/logger"
	"github.com/patstar123/go-base"
	bu "github.com/patstar123/go-base/utils"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var clients = make(map[string]*Client)
var lock = sync.Mutex{}
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var config _Config

type Client struct {
	id       string
	ws       *websocket.Conn
	messages chan []byte
}

func main() {
	loadBuilding()

	res := loadConfigFile("config.yaml")
	if !res.IsOk() {
		logger.Warnw("loadConfigFile failed", res)
		return
	}

	base.InitLogger("msg", &config.Logging)
	logger.Infow("loadConfigFile", "config", config)

	r := mux.NewRouter()
	r.HandleFunc("/sub_msg", handleConnections)
	r.HandleFunc("/reset_remind", handleResetRemind).Methods("POST", "GET")

	http.Handle("/", r)
	err := http.ListenAndServe(":"+config.ApiPort, nil)
	if err != nil {
		logger.Warnw("http.ListenAndServe failed", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warnw("Upgrade websocket failed", err)
		return
	}
	defer ws.Close()

	var clientId string
	err = ws.ReadJSON(&clientId)
	if err != nil {
		logger.Warnw("Error reading client ID:", err)
		return
	}

	logger.Infof("Client %s connected", clientId)

	client := &Client{
		id:       clientId,
		ws:       ws,
		messages: make(chan []byte, 10),
	}

	lock.Lock()
	clients[clientId] = client
	lock.Unlock()

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			closeWs(client)
			break
		}

		if messageType == websocket.TextMessage {
			if len(client.messages) == 10 {
				logger.Warnw("too many message", nil)
				closeWs(client)
				break
			}
			client.messages <- message
		}
	}
}

func handleResetRemind(w http.ResponseWriter, r *http.Request) {
	clientIds, ok := r.URL.Query()["clientId"]
	if !ok || len(clientIds) == 0 {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	clientId := clientIds[0]
	lock.Lock()
	client, ok := clients[clientId]
	lock.Unlock()

	if !ok {
		http.Error(w, "Client not connected", http.StatusNotFound)
		return
	}

	err := client.ws.WriteMessage(websocket.TextMessage, []byte("reset_remind"))
	if err != nil {
		http.Error(w, "Failed to send message to client", http.StatusInternalServerError)
		return
	}

	message, ok := <-client.messages
	if !ok {
		http.Error(w, "Failed to read response from client", http.StatusInternalServerError)
		return
	}

	w.Write(message)
}

func closeWs(client *Client) {
	logger.Infof("Client %s disconnected", client.id)
	lock.Lock()
	delete(clients, client.id)
	lock.Unlock()
	close(client.messages)
	client.ws.Close()
}

type _Config struct {
	ApiPort string        `yaml:"api_port" json:"apiPort"`
	Logging logger.Config `yaml:"logging,omitempty" json:"-"`
}

func loadConfigFile(configFile string) base.Result {
	res := bu.GetConfig(configFile, &config)
	if !res.IsOk() {
		return res
	}

	if config.ApiPort == "" {
		config.ApiPort = "28081"
	}

	return base.SUCCESS
}
