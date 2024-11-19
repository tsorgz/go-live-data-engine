package services

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
	"tsorgz/live-data-engine/internal/models"
)

type CSVWatcher struct {
	filePath string
	offset   int64
	data     chan models.Note
	mu       sync.RWMutex
	status   chan ServiceStatus
}

func NewCSVWatcher(filePath string) *CSVWatcher {
	return &CSVWatcher{
		filePath: filePath,
		data:     make(chan models.Note, 10000),
		status:   make(chan ServiceStatus, 1),
	}
}

func (w *CSVWatcher) Start(ctx context.Context) {
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
			if err := w.checkNewData(ctx); err != nil {
				w.status <- ServiceStatus{IsHealthy: false, Error: err}
				time.Sleep(time.Second) // Backoff on error
			} else {
				w.status <- ServiceStatus{IsHealthy: true}
			}
		}
	}
}

func (w *CSVWatcher) checkNewData(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := os.Open(w.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(w.offset, 0); err != nil {
		return err
	}

	reader := csv.NewReader(bufio.NewReaderSize(file, 32*1024))
	reader.FieldsPerRecord = 4
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Bad Record: %+q, length: %d", record, len(record))
				return err
			}

			timestamp, err := strconv.ParseInt(record[0], 10, 64)
			if err != nil {
				continue // Skip invalid records
			}
			userID, err := strconv.Atoi(record[1])
			if err != nil {
				continue
			}

			note := models.Note{
				Timestamp: timestamp,
				UserID:    userID,
				Note:      record[2],
			}

			select {
			case w.data <- note:
			default:
				log.Println("Warning: CSV channel buffer full, skipping record")
			}
			newOffset, _ := file.Seek(0, 1)
			w.offset = newOffset
		}
	}

}

func (w *CSVWatcher) GetDataChannel() <-chan models.Note {
	return w.data
}

func (w *CSVWatcher) GetStatus() <-chan ServiceStatus {
	return w.status
}
