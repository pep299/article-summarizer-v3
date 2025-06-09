package main

import (
	"log"

	// cloud_function.goのinit()を実行したいので
	_ "github.com/pep299/article-summarizer-v3"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	if err := funcframework.Start("8080"); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}