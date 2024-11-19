package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"tsorgz/live-data-engine/internal/models"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	data       chan []models.Task
	status     chan ServiceStatus
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		data:   make(chan []models.Task, 1000),
		status: make(chan ServiceStatus, 1),
	}
}

func (c *APIClient) StreamData(ctx context.Context, userId int) {
	frequencyInMs, err := strconv.Atoi(os.Getenv("TICK_FREQUENCY"))
	if err != nil {
		log.Panicf("Env TICK_FREQUENCY Error %v", err)
	}

	ticker := time.NewTicker(time.Duration(frequencyInMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.fetchData(ctx, userId); err != nil {
				c.status <- ServiceStatus{IsHealthy: false, Error: err}
			} else {
				c.status <- ServiceStatus{IsHealthy: true}
			}
		}
	}

}

func (c *APIClient) fetchData(ctx context.Context, userId int) error {
	// Send request to external service
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?user_id=%d", c.baseURL, userId), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var tasks []models.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return err
	}

	latestTasks := tasks
	if len(tasks) > 10 {
		latestTasks = tasks[len(tasks)-10:]
	}

	select {
	case c.data <- latestTasks:
	default:
		select {
		// Push head out if arrayuy is full
		case <-c.data:
			select {
			case c.data <- latestTasks:
			default:
				log.Println("Warning: API channel buffer failed to enqueue data")
			}
		default:
		}
	}

	return nil
}

func (c *APIClient) GetDataChannel() <-chan []models.Task {
	return c.data
}

func (c *APIClient) GetStatus() <-chan ServiceStatus {
	return c.status
}
