package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Enemy struct {
	Geometry [][]int `json:"geometry"`
	Status   string  `json:"status"`
	Kills    int     `json:"kills"`
}

// Исходные враги
var enemies = []Enemy{
	{Geometry: [][]int{{35, 81, 4}}, Status: "alive", Kills: 0},
	{Geometry: [][]int{{69, 60, 2}}, Status: "alive", Kills: 0},
	{Geometry: [][]int{{175, 118, 58}}, Status: "alive", Kills: 0},
	{Geometry: [][]int{{14, 71, 42}}, Status: "alive", Kills: 0},
	{Geometry: [][]int{{58, 74, 7}, {58, 75, 7}, {58, 76, 7}, {58, 77, 7}}, Status: "alive", Kills: 0},
}

var mapSize = [3]int{180, 180, 60} // Размер карты

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var mutex = &sync.Mutex{}

func moveEnemies() {
	for i := range enemies {
		for j := range enemies[i].Geometry {
			enemies[i].Geometry[j][0]++ // Двигаем врага вперед по оси X

			// Проверяем границы карты
			if enemies[i].Geometry[j][0] >= mapSize[0] {
				enemies[i].Geometry[j][0] = 0 // Перемещаем в начало
			}
		}
	}
}

func broadcastEnemies() {
	mutex.Lock()
	defer mutex.Unlock()

	data, _ := json.Marshal(enemies)

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
	http.HandleFunc("/ws", handleConnections)

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			moveEnemies()
			broadcastEnemies()
		}
	}()

	log.Println("WebSocket сервер запущен на :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Ошибка сервера:", err)
	}
}
