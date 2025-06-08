package model

type Summary struct {
	Content       string  `json:"content"`
	KeyPoints     []string `json:"key_points"`
	TechnicalTags []string `json:"technical_tags"`
	Sentiment     string   `json:"sentiment"`
	Difficulty    string   `json:"difficulty"`
	Relevance     float64  `json:"relevance"`
}

type ArticleSummary struct {
	Article Article `json:"article"`
	Summary Summary `json:"summary"`
}