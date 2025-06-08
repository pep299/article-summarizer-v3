package server

import (
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handler"
	"github.com/pep299/article-summarizer-v3/internal/model"
	"github.com/pep299/article-summarizer-v3/internal/router"
	"github.com/pep299/article-summarizer-v3/internal/service"
)

type App struct {
	feedService *service.FeedService
	urlService  *service.URLService
}

type Server struct {
	app    *App
	router router.Router
	config *config.Config
}

func NewServer(cfg *config.Config) (*Server, error) {
	model.Feeds["hatena"].URL = cfg.HatenaRSSURL
	model.Feeds["lobsters"].URL = cfg.LobstersRSSURL

	// TODO: Initialize repositories with actual implementations
	// For now, we'll use nil to maintain compatibility
	feedService := service.NewFeedService(nil, nil, nil, nil)
	urlService := service.NewURLService(nil, nil)

	app := &App{
		feedService: feedService,
		urlService:  urlService,
	}

	webhookHandler := handler.NewWebhookHandler(urlService)
	processHandler := handler.NewProcessHandler(feedService)

	httpRouter := router.NewHTTPRouter(webhookHandler, processHandler, cfg)
	httpRouter.SetupRoutes()

	return &Server{
		app:    app,
		router: httpRouter,
		config: cfg,
	}, nil
}

func (s *Server) Start() error {
	return s.router.Run()
}

func (s *Server) ProcessSingleFeed(feedName string) error {
	// For backwards compatibility - delegate to the old handlers package
	// TODO: This should use s.app.feedService.ProcessFeed() once repositories are implemented
	return fmt.Errorf("not implemented - use handlers.Server for now")
}

func (s *Server) ProcessSingleURL(url string) error {
	// For backwards compatibility - delegate to the old handlers package
	// TODO: This should use s.app.urlService.ProcessURL() once repositories are implemented
	return fmt.Errorf("not implemented - use handlers.Server for now")
}