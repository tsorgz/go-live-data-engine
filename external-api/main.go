package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Task struct {
	Timestamp int64  `json:"timestamp"`
	UserID    int    `json:"user_id"`
	Task      string `json:"task"`
}

var tasks [1000][]Task
var mu sync.RWMutex

func main() {
	go addData()
	r := mux.NewRouter()
	r.HandleFunc("/data", getData).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func addData() {
	for {
		time.Sleep(time.Microsecond * time.Duration(rand.Intn(499990)+10))
		userId := rand.Intn(1000)
		newTask := Task{
			Timestamp: time.Now().UnixMilli(),
			UserID:    userId + 1,
			Task:      fmt.Sprintf("Task %d", rand.Int()),
		}

		mu.Lock()
		tasks[userId] = append(tasks[userId], newTask)
		mu.Unlock()
		// log.Printf("Data Added: %+v", newTask)
	}
}

func getData(w http.ResponseWriter, r *http.Request) {

	userId := r.URL.Query().Get("user_id")
	userIdInt, err := strconv.Atoi(userId)
	if err != nil || userIdInt < 1 || userIdInt > 1000 {
		http.Error(w, "Bad User ID", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	mu.RLock()
	err = json.NewEncoder(w).Encode(tasks[userIdInt-1])
	mu.RUnlock()
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %v", err)
		return
	}

}
