package models

// Task is the resource. Struct tags control the JSON field names.
type Event struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Description  string   `json:"description"`
}