package model

import "time"

type Article struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	PubDate     time.Time `json:"pub_date"`
	GUID        string    `json:"guid"`
}