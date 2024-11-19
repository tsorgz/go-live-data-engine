package models

type User struct {
	ID   int    `json:"user_id"`
	Name string `json:"user_name"`
}

type Note struct {
	Timestamp int64  `json:"timestamp"`
	UserID    int    `json:"user_id"`
	Note      string `json:"note"`
}

type Task struct {
	Timestamp int64  `json:"timestamp"`
	UserID    int    `json:"user_id"`
	Task      string `json:"task"`
}

type UserStream struct {
	UserID   int           `json:"user_id"`
	UserName string        `json:"user_name"`
	Tasks    []TimedString `json:"tasks"`
	Notes    []TimedString `json:"notes"`
}

type TimedString struct {
	Timestamp int64  `json:"timestamp"`
	Content   string `json:"content"`
}
