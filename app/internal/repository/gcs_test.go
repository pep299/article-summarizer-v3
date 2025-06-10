package repository

import (
	"testing"
	"time"
)

func TestGCSRepository_GenerateKey(t *testing.T) {
	repo := &gcsRepository{}

	tests := []struct {
		name     string
		article  Item
		expected string
	}{
		{
			name: "URL with GUID",
			article: Item{
				GUID: "https://example.com/article/123",
				Link: "https://example.com/article/123?utm_source=test",
			},
			expected: "https://example.com/article/123",
		},
		{
			name: "URL without GUID",
			article: Item{
				GUID: "",
				Link: "https://example.com/article/456?param=value",
			},
			expected: "https://example.com/article/456",
		},
		{
			name: "URL with fragment",
			article: Item{
				GUID: "",
				Link: "https://example.com/article/789#section1",
			},
			expected: "https://example.com/article/789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.GenerateKey(tt.article)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGCSRepository_IsProcessed(t *testing.T) {
	repo := &gcsRepository{}

	// テスト用インデックス作成
	index := map[string]*IndexEntry{
		"https://example.com/processed": {
			Title:         "Processed Article",
			URL:           "https://example.com/processed",
			Source:        "test",
			PubDate:       time.Now(),
			ProcessedDate: time.Now(),
		},
	}

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "existing article",
			key:      "https://example.com/processed",
			expected: true,
		},
		{
			name:     "non-existing article",
			key:      "https://example.com/new",
			expected: false,
		},
		{
			name:     "empty key",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.IsProcessed(tt.key, index)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGCSRepository_NormalizeURL(t *testing.T) {
	repo := &gcsRepository{}

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "URL with query parameters",
			input:    "https://example.com/article?utm_source=test&param=value",
			expected: "https://example.com/article",
			hasError: false,
		},
		{
			name:     "URL with fragment",
			input:    "https://example.com/article#section1",
			expected: "https://example.com/article",
			hasError: false,
		},
		{
			name:     "URL with both query and fragment",
			input:    "https://example.com/article?param=value#section1",
			expected: "https://example.com/article",
			hasError: false,
		},
		{
			name:     "clean URL",
			input:    "https://example.com/article",
			expected: "https://example.com/article",
			hasError: false,
		},
		{
			name:     "invalid URL",
			input:    "://invalid-url",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.normalizeURL(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func TestGCSRepository_DuplicateCheckWorkflow(t *testing.T) {
	// 重複チェックの統合ワークフローテスト
	repo := &gcsRepository{}

	// テスト記事
	article1 := Item{
		Title: "Test Article 1",
		Link:  "https://example.com/article/1?utm_source=rss",
		GUID:  "https://example.com/article/1",
	}

	article2 := Item{
		Title: "Test Article 2",
		Link:  "https://example.com/article/1?utm_source=twitter", // 同じ記事、異なるパラメータ
		GUID:  "https://example.com/article/1",
	}

	article3 := Item{
		Title: "Test Article 3",
		Link:  "https://example.com/article/2",
		GUID:  "",
	}

	// 空のインデックスから開始
	index := make(map[string]*IndexEntry)

	// 1. 記事1をチェック（未処理）
	key1 := repo.GenerateKey(article1)
	if repo.IsProcessed(key1, index) {
		t.Error("Article 1 should not be processed initially")
	}

	// 2. 記事1を処理済みとしてマーク（シミュレート）
	index[key1] = &IndexEntry{
		Title:         article1.Title,
		URL:           key1,
		Source:        "test",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
	}

	// 3. 記事2をチェック（同じ記事なので処理済みのはず）
	key2 := repo.GenerateKey(article2)
	if !repo.IsProcessed(key2, index) {
		t.Error("Article 2 should be processed (same as article 1)")
	}

	// 4. キーが同じことを確認
	if key1 != key2 {
		t.Errorf("Keys should be the same: key1='%s', key2='%s'", key1, key2)
	}

	// 5. 記事3をチェック（未処理）
	key3 := repo.GenerateKey(article3)
	if repo.IsProcessed(key3, index) {
		t.Error("Article 3 should not be processed")
	}

	// 6. キーが異なることを確認
	if key1 == key3 {
		t.Error("Key1 and Key3 should be different")
	}

	t.Logf("✅ Duplicate check workflow test passed")
	t.Logf("Key1 (article1): %s", key1)
	t.Logf("Key2 (article2): %s", key2)
	t.Logf("Key3 (article3): %s", key3)
}
