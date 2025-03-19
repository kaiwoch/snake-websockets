package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GameStruct struct {
	MapSize []int   `json:"mapSize"`
	Name    string  `json:"name"`
	Points  int     `json:"points"`
	Fences  [][]int `json:"fences"`
	Snakes  []struct {
		ID             string  `json:"id"`
		Direction      []int   `json:"direction"`
		OldDirection   []int   `json:"oldDirection"`
		Geometry       [][]int `json:"geometry"`
		DeathCount     int     `json:"deathCount"`
		Status         string  `json:"status"`
		ReviveRemainMs int     `json:"reviveRemainMs"`
	} `json:"snakes"`
	Enemies []Enemy `json:"enemies"`
	Food    []struct {
		C      []int `json:"c"`
		Points int   `json:"points"`
		Type   int   `json:"type"`
	} `json:"food"`
	SpecialFood struct {
		Golden     [][]int `json:"golden"`
		Suspicious [][]int `json:"suspicious"`
	} `json:"specialFood"`
	Turn             int           `json:"turn"`
	ReviveTimeoutSec int           `json:"reviveTimeoutSec"`
	TickRemainMs     int           `json:"tickRemainMs"`
	Errors           []interface{} `json:"errors"`
}

type Enemy struct {
	Geometry [][]int `json:"geometry"`
	Status   string  `json:"status"`
	Kills    int     `json:"kills"`
}

var (
	gs      = GameStruct{}
	mapSize = [3]int{180, 180, 60}
	enemies []Enemy
	mutex   = &sync.Mutex{}
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)

func ReadLog(filename string) ([]Enemy, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&gs); err != nil {
		return nil, err
	}
	return gs.Enemies, nil
}

func moveEnemies() {
	mutex.Lock()
	defer mutex.Unlock()

	for i := range enemies {
		headX, headY, headZ := enemies[i].Geometry[0][0], enemies[i].Geometry[0][1], enemies[i].Geometry[0][2]
		headX++

		if headX >= mapSize[0] {
			headX = 0
		}

		for j := len(enemies[i].Geometry) - 1; j > 0; j-- {
			enemies[i].Geometry[j] = enemies[i].Geometry[j-1]
		}

		enemies[i].Geometry[0] = []int{headX, headY, headZ}
	}
}

func broadcastEnemies() {
	mutex.Lock()
	defer mutex.Unlock()

	data, _ := json.Marshal(gs)

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			break
		}
	}
}

func main() {
	var err error
	enemies, err = ReadLog("test_logs_2.json")
	if err != nil {
		log.Fatal("Ошибка загрузки змей:", err)
	}

	http.HandleFunc("/ws", handleConnections)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	go func() {
		for {
			time.Sleep(time.Second)
			moveEnemies()
			broadcastEnemies()
		}
	}()

	log.Println("Сервер запущен на :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Ошибка сервера:", err)
	}
}
