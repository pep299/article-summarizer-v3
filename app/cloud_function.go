package app

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"

	"github.com/pep299/article-summarizer-v3/internal/transport/server"
)

func init() {
	// 環境変数から関数名を取得（必須）
	functionTarget := os.Getenv("FUNCTION_TARGET")
	if functionTarget == "" {
		log.Fatal("❌ Error: FUNCTION_TARGET environment variable is not set")
	}

	log.Printf("✅ Registering function: %s", functionTarget)

	// 関数を登録
	functions.HTTP(functionTarget, server.HandleRequest)
}
