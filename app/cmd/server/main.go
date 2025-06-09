package main

import (
	"log"

	// cloud_function.goのinit()を実行したいので
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	_ "github.com/pep299/article-summarizer-v3"
)

func main() {
	if err := funcframework.Start("8080"); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
