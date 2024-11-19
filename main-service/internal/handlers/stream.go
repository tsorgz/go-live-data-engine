package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"tsorgz/live-data-engine/internal/models"
	"tsorgz/live-data-engine/internal/services"
)

type UserDataAggregator struct {
	mu    sync.RWMutex
	users map[int]*models.UserStream
}

type TimedData struct {
	timestamp int64
	content   string
}

func NewUserDataAggregator() *UserDataAggregator {
	return &UserDataAggregator{
		users: make(map[int]*models.UserStream),
	}
}

func (a *UserDataAggregator) AddUser(user models.User) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.users[user.ID] = &models.UserStream{
		UserID:   user.ID,
		UserName: user.Name,
		Tasks:    make([]models.TimedString, 0, 10),
		Notes:    make([]models.TimedString, 0, 10),
	}
}

func (a *UserDataAggregator) CompareRecentTask(userID int, task models.Task) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return len(a.users[userID].Tasks) < 1 || (task.Timestamp != a.users[userID].Tasks[0].Timestamp && task.Task != a.users[userID].Tasks[0].Content)
}

func (a *UserDataAggregator) AddTasks(userID int, tasks []models.Task) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if userData, exists := a.users[userID]; exists {
		userData.Tasks = nil
		// Load tasks backwards from array
		for i := len(tasks) - 1; i >= 0; i-- {
			task := tasks[i]
			timedTask := models.TimedString{
				Timestamp: task.Timestamp,
				Content:   task.Task,
			}
			userData.Tasks = append(userData.Tasks, timedTask)
		}
		// Keep only most recent 10 tasks
		if len(userData.Tasks) > 10 {
			userData.Tasks = userData.Tasks[len(userData.Tasks)-10:]
		}
	}
}

func (a *UserDataAggregator) AddNote(userID int, timestamp int64, note string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if userData, exists := a.users[userID]; exists {
		timedNote := models.TimedString{
			Timestamp: timestamp,
			Content:   note,
		}
		userData.Notes = append(userData.Notes, timedNote)
		// Keep only most recent 10 notes
		if len(userData.Notes) > 10 {
			userData.Notes = userData.Notes[len(userData.Notes)-10:]
		}
	}
}

func (a *UserDataAggregator) GetUserStream(userID int) *models.UserStream {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if userData, exists := a.users[userID]; exists {
		return userData
	}
	return nil
}

func StreamHandler(db *sql.DB, csvWatcher *services.CSVWatcher, apiClient *services.APIClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
		if err != nil {
			http.Error(w, "Invalid user_id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Monitor connection closure
		go func() {
			<-r.Context().Done()
			cancel()
		}()

		aggregator := NewUserDataAggregator()

		// Load user data from database
		row := db.QueryRowContext(ctx, "SELECT id, name FROM users WHERE id = $1", userID)
		var user models.User
		if err := row.Scan(&user.ID, &user.Name); err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		aggregator.AddUser(user)

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		var wg sync.WaitGroup

		// Process CSV data
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case status := <-csvWatcher.GetStatus():
					if !status.IsHealthy {
						log.Printf("CSV service unhealthy: %v", status.Error)
					}
				case note := <-csvWatcher.GetDataChannel():
					if note.UserID == userID {
						aggregator.AddNote(userID, note.Timestamp, note.Note)
						userData := aggregator.GetUserStream(userID)
						if err := encoder.Encode(userData); err != nil {
							cancel()
							return
						}
						flusher.Flush()
					}
				}
			}
		}()

		// Process API data
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case status := <-apiClient.GetStatus():
					if !status.IsHealthy {
						log.Printf("API service unhealthy: %v", status.Error)
					}
				case tasks := <-apiClient.GetDataChannel():
					if len(tasks) > 0 && tasks[0].UserID == userID && aggregator.CompareRecentTask(userID, tasks[len(tasks)-1]) {
						aggregator.AddTasks(userID, tasks)
						userData := aggregator.GetUserStream(userID)
						if err := encoder.Encode(userData); err != nil {
							cancel()
							return
						}
						flusher.Flush()
					}
				}
			}
		}()

		// Start services
		go csvWatcher.Start(ctx)
		go apiClient.StreamData(ctx, userID)

		wg.Wait()
	}
}
