package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type MapStruct struct {
	Enemies []Enemy `json:"enemies"`
}

type Enemy struct {
	Geometry [][]int `json:"geometry"`
}

var (
	mapSize = [3]int{180, 180, 60} // Размер карты
	enemies []Enemy                // Глобальная переменная с врагами
	mutex   = &sync.Mutex{}        // Мьютекс для потокобезопасного доступа
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)

// Читаем змей из файла JSON
func ReadLog(filename string) ([]Enemy, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sm MapStruct
	if err := json.NewDecoder(f).Decode(&sm); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	return sm.Enemies, nil
}

// Двигаем всех врагов
func moveEnemies() {
	mutex.Lock()
	defer mutex.Unlock()

	for i := range enemies {
		// Берем координаты головы змеи
		headX, headY, headZ := enemies[i].Geometry[0][0], enemies[i].Geometry[0][1], enemies[i].Geometry[0][2]

		// Двигаем голову вперёд (здесь можно выбрать любую логику движения)
		headX++ // Например, двигаем вправо по оси X

		// Проверяем границы карты
		if headX >= mapSize[0] {
			headX = 0 // Перемещаем в начало
		}

		// Двигаем тело змеи: каждый сегмент занимает место предыдущего
		for j := len(enemies[i].Geometry) - 1; j > 0; j-- {
			enemies[i].Geometry[j] = enemies[i].Geometry[j-1] // Текущий сегмент занимает позицию предыдущего
		}

		// Обновляем позицию головы
		enemies[i].Geometry[0] = []int{headX, headY, headZ}
	}
}

// Отправляем обновленные координаты всем клиентам
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

// Обработка WebSocket-подключений
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка WebSocket:", err)
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
		log.Fatal("[ERROR] Не удалось загрузить змей из JSON:", err)
	}

	http.HandleFunc("/ws", handleConnections)

	// Запускаем бесконечный цикл движения и обновления
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			moveEnemies()
			broadcastEnemies()
		}
	}()

	log.Println("WebSocket сервер запущен на :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Ошибка сервера:", err)
	}
}
