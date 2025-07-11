package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

// IndexEntry represents a processed item in the index
type IndexEntry struct {
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	Source        string    `json:"source"`
	PubDate       time.Time `json:"pub_date"`
	ProcessedDate time.Time `json:"processed_date"`
}

// ProcessedArticleRepository manages processed articles index for Cloud Function
type ProcessedArticleRepository interface {
	LoadIndex(ctx context.Context) (map[string]*IndexEntry, error)
	IsProcessed(key string, index map[string]*IndexEntry) bool
	MarkAsProcessed(ctx context.Context, article Item) error
	GenerateKey(article Item) string
	Close() error
}

type gcsRepository struct {
	client     *storage.Client
	bucketName string
	indexFile  string
}

const defaultIndexFileName = "index-v2.json"

// NewProcessedArticleRepository creates a new processed article repository
func NewProcessedArticleRepository() (ProcessedArticleRepository, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}

	// Get bucket name from environment
	bucketName := "article-summarizer-processed-articles"
	if env := os.Getenv("CACHE_BUCKET"); env != "" {
		bucketName = env
	}

	// Get index file name from environment (for testing)
	indexFileName := defaultIndexFileName
	if env := os.Getenv("CACHE_INDEX_FILE"); env != "" {
		indexFileName = env
	}

	return &gcsRepository{
		client:     client,
		bucketName: bucketName,
		indexFile:  indexFileName,
	}, nil
}

// LoadIndex loads the index from GCS
func (g *gcsRepository) LoadIndex(ctx context.Context) (map[string]*IndexEntry, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	obj := g.client.Bucket(g.bucketName).Object(g.indexFile)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			// Index doesn't exist yet, return empty index
			return make(map[string]*IndexEntry), nil
		}
		logger.Printf("Error opening GCS index reader: %v\nStack:\n%s", err, debug.Stack())
		return nil, fmt.Errorf("opening index reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		logger.Printf("Error reading GCS index data: %v\nStack:\n%s", err, debug.Stack())
		return nil, fmt.Errorf("reading index data: %w", err)
	}

	var index map[string]*IndexEntry
	if err := json.Unmarshal(data, &index); err != nil {
		logger.Printf("Error unmarshaling GCS index: %v", err)
		return nil, fmt.Errorf("unmarshaling index: %w", err)
	}

	return index, nil
}

// SaveIndex saves the index to GCS
func (g *gcsRepository) saveIndex(ctx context.Context, index map[string]*IndexEntry) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	data, err := json.Marshal(index)
	if err != nil {
		logger.Printf("Error marshaling GCS index: %v", err)
		return fmt.Errorf("marshaling index: %w", err)
	}

	obj := g.client.Bucket(g.bucketName).Object(g.indexFile)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		logger.Printf("Error writing GCS index data: %v\nStack:\n%s", err, debug.Stack())
		return fmt.Errorf("writing index data: %w", err)
	}

	if err := writer.Close(); err != nil {
		logger.Printf("Error closing GCS index writer: %v\nStack:\n%s", err, debug.Stack())
		return fmt.Errorf("closing index writer: %w", err)
	}

	return nil
}

// IsProcessed checks if an article is already processed using the startup index
func (g *gcsRepository) IsProcessed(key string, index map[string]*IndexEntry) bool {
	_, exists := index[key]
	return exists
}

// MarkAsProcessed marks an article as processed (includes GCS re-fetch and update)
func (g *gcsRepository) MarkAsProcessed(ctx context.Context, article Item) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	// 1. Load latest index from GCS (to handle concurrent updates)
	index, err := g.LoadIndex(ctx)
	if err != nil {
		logger.Printf("Error loading latest GCS index for marking processed: %v", err)
		return fmt.Errorf("loading latest index: %w", err)
	}

	// 2. Add processed item
	key := g.GenerateKey(article)
	index[key] = &IndexEntry{
		Title:         article.Title,
		URL:           key, // Normalized URL
		Source:        article.Source,
		PubDate:       article.ParsedDate,
		ProcessedDate: time.Now(),
	}

	// 3. Save updated index to GCS
	if err := g.saveIndex(ctx, index); err != nil {
		logger.Printf("Error saving GCS index after marking processed: %v", err)
		return err
	}
	return nil
}

// GenerateKey generates a key for an article
func (g *gcsRepository) GenerateKey(article Item) string {
	// Always use Link for consistent URL-based deduplication
	identifier := article.Link

	// Normalize URL
	normalizedURL, err := g.normalizeURL(identifier)
	if err != nil {
		// Fallback to original identifier if normalization fails
		return strings.TrimSpace(identifier)
	}

	return normalizedURL
}

// Close closes the GCS client
func (g *gcsRepository) Close() error {
	if err := g.client.Close(); err != nil {
		log.Printf("Error closing GCS client: %v", err)
		return err
	}
	return nil
}

// normalizeURL normalizes URL for consistent duplicate detection
func (g *gcsRepository) normalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	// 1. Force HTTPS protocol
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	// 2. Remove www subdomain
	if strings.HasPrefix(parsedURL.Host, "www.") {
		parsedURL.Host = parsedURL.Host[4:]
	}

	// 3. Remove query parameters and fragments
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	// 4. Remove trailing slash from path (except root)
	if parsedURL.Path != "/" && strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	}

	// 5. Convert host to lowercase
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	// 6. Convert path to lowercase for case-insensitive comparison
	parsedURL.Path = strings.ToLower(parsedURL.Path)

	return parsedURL.String(), nil
}
