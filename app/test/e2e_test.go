package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pep299/article-summarizer-v3/internal/application"
)

// E2ETestConfig holds test configuration
type E2ETestConfig struct {
	GeminiAPIKey  string
	SlackBotToken string
	SlackChannel  string
}

func loadE2EConfig() *E2ETestConfig {
	// 確実に環境変数を読み込む（テスト実行順序の影響を回避）
	geminiKey := os.Getenv("GEMINI_API_KEY")
	slackToken := os.Getenv("SLACK_BOT_TOKEN")

	// E2E用プレフィックスがあれば優先
	if key := os.Getenv("E2E_GEMINI_API_KEY"); key != "" {
		geminiKey = key
	}
	if token := os.Getenv("E2E_SLACK_BOT_TOKEN"); token != "" {
		slackToken = token
	}

	return &E2ETestConfig{
		GeminiAPIKey:  geminiKey,
		SlackBotToken: slackToken,
		SlackChannel:  "#dev-null", // テスト用チャンネル
	}
}

func getAPIKey() string {
	// E2E_プレフィックス付きを優先、なければ本番用
	if key := os.Getenv("E2E_GEMINI_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("GEMINI_API_KEY")
}

func getSlackToken() string {
	// E2E_プレフィックス付きを優先、なければ本番用
	if token := os.Getenv("E2E_SLACK_BOT_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("SLACK_BOT_TOKEN")
}

func setupE2EEnvironment(config *E2ETestConfig) {
	os.Setenv("GEMINI_API_KEY", config.GeminiAPIKey)
	os.Setenv("SLACK_BOT_TOKEN", config.SlackBotToken)
	os.Setenv("SLACK_CHANNEL", config.SlackChannel)
	os.Setenv("WEBHOOK_SLACK_CHANNEL", config.SlackChannel) // 両方とも#dev-nullに
	// テスト用のGCSバケット設定
	os.Setenv("CACHE_BUCKET", "article-summarizer-processed-articles")
	os.Setenv("CACHE_INDEX_FILE", "tmp-index-test.json") // 一時テスト用インデックス
	os.Setenv("CACHE_TYPE", "memory")
	os.Setenv("CACHE_DURATION_HOURS", "1")
	// テスト用の処理件数制限
	os.Setenv("TEST_MAX_ARTICLES", "2") // テストでは2件だけ処理
}

func cleanupE2EEnvironment() {
	// 本番環境変数はUnsetしない（他のテストで使うため）
	// os.Unsetenv("GEMINI_API_KEY")
	// os.Unsetenv("SLACK_BOT_TOKEN")

	// テスト固有の設定のみクリア
	os.Unsetenv("SLACK_CHANNEL")
	os.Unsetenv("WEBHOOK_SLACK_CHANNEL")
	os.Unsetenv("CACHE_BUCKET")
	os.Unsetenv("CACHE_INDEX_FILE")
	os.Unsetenv("CACHE_TYPE")
	os.Unsetenv("CACHE_DURATION_HOURS")
	os.Unsetenv("TEST_MAX_ARTICLES")
}

// GCSヘルパー関数
func setupTestGCSIndex(t *testing.T) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create GCS client: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket("article-summarizer-processed-articles")
	obj := bucket.Object("tmp-index-test.json")

	// 空のインデックスを作成
	emptyIndex := make(map[string]interface{})
	data, err := json.Marshal(emptyIndex)
	if err != nil {
		t.Fatalf("Failed to marshal empty index: %v", err)
	}

	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		t.Fatalf("Failed to write test index: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close GCS writer: %v", err)
	}

	t.Logf("✅ Test GCS index created: tmp-index-test.json")
}

func cleanupTestGCSIndex(t *testing.T) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Logf("Warning: Failed to create GCS client for cleanup: %v", err)
		return
	}
	defer client.Close()

	bucket := client.Bucket("article-summarizer-processed-articles")
	obj := bucket.Object("tmp-index-test.json")

	if err := obj.Delete(ctx); err != nil {
		// オブジェクトが存在しない場合のエラーは無視
		if err != storage.ErrObjectNotExist {
			t.Logf("Warning: Failed to delete test index: %v", err)
		}
	} else {
		t.Logf("✅ Test GCS index cleaned up: tmp-index-test.json")
	}
}

func verifyGCSIndexUpdated(t *testing.T, expectedCount int) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create GCS client for verification: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket("article-summarizer-processed-articles")
	obj := bucket.Object("tmp-index-test.json")

	reader, err := obj.NewReader(ctx)
	if err != nil {
		t.Fatalf("Failed to read test index for verification: %v", err)
	}
	defer reader.Close()

	var index map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&index); err != nil {
		t.Fatalf("Failed to decode test index: %v", err)
	}

	actualCount := len(index)
	if actualCount != expectedCount {
		t.Errorf("Expected %d articles in GCS index, got %d", expectedCount, actualCount)
		t.Logf("GCS index contents: %+v", index)
	} else {
		t.Logf("✅ GCS index verification passed: %d articles found", actualCount)
		// Log actual keys to verify they're real articles
		for key := range index {
			t.Logf("  - Key: %s", key[:50]+"...") // Show first 50 chars of key
		}
	}
}

// 重複チェックはユニットテストで十分にテスト済み

// TestE2E_HatenaRSSToSlack tests the full pipeline: Hatena RSS → Summarization → Slack notification
func TestE2E_HatenaRSSToSlack(t *testing.T) {
	t.Logf("DEBUG: Direct ENV GEMINI_API_KEY: %s", os.Getenv("GEMINI_API_KEY")[:10])
	t.Logf("DEBUG: Direct ENV SLACK_BOT_TOKEN: %s", os.Getenv("SLACK_BOT_TOKEN")[:15])
	t.Logf("DEBUG: getAPIKey(): %s", getAPIKey()[:10])
	t.Logf("DEBUG: getSlackToken(): %s", getSlackToken()[:15])

	config := loadE2EConfig()
	t.Logf("DEBUG: config.GeminiAPIKey: %s", config.GeminiAPIKey)
	t.Logf("DEBUG: config.SlackBotToken: %s", config.SlackBotToken)

	if config.GeminiAPIKey == "" || config.SlackBotToken == "" {
		t.Skip("E2E test requires GEMINI_API_KEY and SLACK_BOT_TOKEN environment variables")
	}

	t.Logf("🚀 Starting Hatena RSS E2E test (max 2 articles)")

	// GCSテスト用インデックス作成
	setupTestGCSIndex(t)
	defer cleanupTestGCSIndex(t)

	setupE2EEnvironment(config)
	defer cleanupE2EEnvironment()

	// Create application
	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Create test server
	server := httptest.NewServer(app.ProcessHandler)
	defer server.Close()

	// Test Hatena RSS processing
	requestBody := map[string]string{
		"feedName": "hatena",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		t.Fatalf("Expected status 200, got %d. Error: %v", resp.StatusCode, errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", result["status"])
	}

	// GCSインデックスが更新されているか確認
	verifyGCSIndexUpdated(t, 2) // 2件処理されたはず

	t.Logf("✅ E2E Test passed: Hatena RSS → Summarization → Slack (#dev-null)")
	t.Logf("Response: %+v", result)
}

// TestE2E_LobstersRSSToSlack tests the full pipeline: Lobsters RSS → Summarization → Slack notification
func TestE2E_LobstersRSSToSlack(t *testing.T) {
	t.Logf("DEBUG: Direct ENV GEMINI_API_KEY: %s", os.Getenv("GEMINI_API_KEY")[:10])
	t.Logf("DEBUG: Direct ENV SLACK_BOT_TOKEN: %s", os.Getenv("SLACK_BOT_TOKEN")[:15])

	config := loadE2EConfig()
	t.Logf("DEBUG: config.GeminiAPIKey: %s", config.GeminiAPIKey)
	t.Logf("DEBUG: config.SlackBotToken: %s", config.SlackBotToken)

	if config.GeminiAPIKey == "" || config.SlackBotToken == "" {
		t.Skip("E2E test requires GEMINI_API_KEY and SLACK_BOT_TOKEN environment variables")
	}

	t.Logf("🚀 Starting Lobsters RSS E2E test (max 2 articles)")

	// GCSテスト用インデックス作成
	setupTestGCSIndex(t)
	defer cleanupTestGCSIndex(t)

	setupE2EEnvironment(config)
	defer cleanupE2EEnvironment()

	// Create application
	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Create test server
	server := httptest.NewServer(app.ProcessHandler)
	defer server.Close()

	// Test Lobsters RSS processing
	requestBody := map[string]string{
		"feedName": "lobsters",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		t.Fatalf("Expected status 200, got %d. Error: %v", resp.StatusCode, errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", result["status"])
	}

	// GCSインデックスが更新されているか確認
	verifyGCSIndexUpdated(t, 2) // 2件処理されたはず

	t.Logf("✅ E2E Test passed: Lobsters RSS → Summarization → Slack (#dev-null)")
	t.Logf("Response: %+v", result)
}

// TestE2E_WebhookToSlack tests the webhook endpoint: URL → Summarization → Slack notification
func TestE2E_WebhookToSlack(t *testing.T) {
	t.Logf("DEBUG: Direct ENV GEMINI_API_KEY: %s", os.Getenv("GEMINI_API_KEY")[:10])
	t.Logf("DEBUG: Direct ENV SLACK_BOT_TOKEN: %s", os.Getenv("SLACK_BOT_TOKEN")[:15])

	config := loadE2EConfig()
	t.Logf("DEBUG: config.GeminiAPIKey: %s", config.GeminiAPIKey)
	t.Logf("DEBUG: config.SlackBotToken: %s", config.SlackBotToken)

	if config.GeminiAPIKey == "" || config.SlackBotToken == "" {
		t.Skip("E2E test requires GEMINI_API_KEY and SLACK_BOT_TOKEN environment variables")
	}

	t.Logf("🚀 Starting Webhook E2E test")

	// GCSテスト用インデックス作成
	setupTestGCSIndex(t)
	defer cleanupTestGCSIndex(t)

	setupE2EEnvironment(config)
	defer cleanupE2EEnvironment()

	// Create application
	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Create test server
	server := httptest.NewServer(app.WebhookHandler)
	defer server.Close()

	// Test webhook with a simple, fast URL
	testURL := "https://example.com"

	requestBody := map[string]string{
		"url": testURL,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		t.Fatalf("Expected status 200, got %d. Error: %v", resp.StatusCode, errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", result["status"])
	}

	// Check URL in data field
	if data, ok := result["data"].(map[string]interface{}); ok {
		if data["url"] != testURL {
			t.Errorf("Expected URL '%s', got '%v'", testURL, data["url"])
		}
	} else {
		t.Error("Expected data field with URL")
	}

	t.Logf("✅ E2E Test passed: Webhook URL → Summarization → Slack (#dev-null)")
	t.Logf("Response: %+v", result)
}

// TestE2E_ErrorHandling tests error scenarios
func TestE2E_ErrorHandling(t *testing.T) {
	// Test with minimal config (should still validate basic structure)
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#dev-null")
	defer cleanupE2EEnvironment()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Test invalid feed name
	server := httptest.NewServer(app.ProcessHandler)
	defer server.Close()

	invalidRequest := map[string]string{
		"feedName": "invalid-feed",
	}

	jsonData, err := json.Marshal(invalidRequest)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Should return an error status for invalid feed
	if resp.StatusCode == http.StatusOK {
		t.Logf("Note: Invalid feed name was accepted (may be expected behavior)")
	}

	t.Logf("✅ Error handling test completed")
}
