package model

type Feed struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
}

var Feeds = map[string]Feed{
	"hatena": {
		Name:        "hatena",
		DisplayName: "はてブ テクノロジー",
		URL:         "", // Will be set from config
	},
	"lobsters": {
		Name:        "lobsters",
		DisplayName: "Lobsters",
		URL:         "", // Will be set from config
	},
}