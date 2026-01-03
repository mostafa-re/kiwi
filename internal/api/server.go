package api

import (
	"kiwi/internal/config"
	"kiwi/internal/models"
	"kiwi/internal/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Server represents the HTTP server
type Server struct {
	app     *fiber.App
	config  *config.Config
	handler *Handler
}

// NewServer creates and configures a new HTTP server
func NewServer(cfg *config.Config, store *storage.ReplicatedStore) *Server {
	handler := NewHandler(store, cfg)

	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: customErrorHandler,
	})

	// Apply middleware
	app.Use(recover.New())
	app.Use(logger.New())

	server := &Server{
		app:     app,
		config:  cfg,
		handler: handler,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.app.Get("/health", s.handler.HealthCheck)

	// Cluster status endpoint
	s.app.Get("/cluster", s.handler.ClusterStatus)

	// API routes group
	api := s.app.Group("/objects")

	api.Put("/", s.handler.PutObject)
	api.Get("/:key", s.handler.GetObject)
	api.Get("/", s.handler.ListObjects)
	api.Delete("/:key", s.handler.DeleteObject)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.app.Listen(":" + s.config.Port)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(models.ErrorResponse{
		Error: err.Error(),
	})
}
