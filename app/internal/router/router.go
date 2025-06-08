package router

import (
	"log"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handler"
	"github.com/pep299/article-summarizer-v3/internal/middleware"
)

type Router interface {
	SetupRoutes()
	Run() error
}

type HTTPRouter struct {
	mux             *http.ServeMux
	webhookHandler  *handler.WebhookHandler
	processHandler  *handler.ProcessHandler
	config          *config.Config
}

func NewHTTPRouter(
	webhookHandler *handler.WebhookHandler,
	processHandler *handler.ProcessHandler,
	config *config.Config,
) *HTTPRouter {
	return &HTTPRouter{
		mux:             http.NewServeMux(),
		webhookHandler:  webhookHandler,
		processHandler:  processHandler,
		config:          config,
	}
}

func (r *HTTPRouter) SetupRoutes() {
	r.mux.HandleFunc("/webhook", r.withAuth(r.webhookHandler.Handle))
	r.mux.HandleFunc("/process", r.withAuth(r.processHandler.Handle))
	r.mux.HandleFunc("/", r.withAuth(r.processHandler.Handle))
}

func (r *HTTPRouter) withAuth(handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		authMiddleware := middleware.Auth(r.config.WebhookAuthToken)
		
		wrappedHandler := authMiddleware(func(c middleware.Context) error {
			return handler(w, req)
		})

		mockContext := &MockContext{req: req, w: w}
		if err := wrappedHandler(mockContext); err != nil {
			log.Printf("Handler error: %v", err)
		}
	}
}

type MockContext struct {
	req *http.Request
	w   http.ResponseWriter
}

func (c *MockContext) Request() *http.Request {
	return c.req
}

func (c *MockContext) JSON(code int, obj interface{}) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(code)
	return nil
}

func (r *HTTPRouter) Run() error {
	log.Println("Server starting on :8080")
	return http.ListenAndServe(":8080", r.mux)
}