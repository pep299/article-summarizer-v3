package app

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/pep299/article-summarizer-v3/internal/server"
)

func init() {
	functions.HTTP("SummarizeArticles", server.HandleRequest)
}