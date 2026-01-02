package api

import (
	"kv-service/internal/models"
	"kv-service/internal/storage"

	"github.com/gofiber/fiber/v2"
)

// Handler contains HTTP request handlers
type Handler struct {
	store storage.Store
}

// NewHandler creates a new handler instance
func NewHandler(store storage.Store) *Handler {
	return &Handler{store: store}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(models.HealthResponse{
		Status: "healthy",
	})
}

// PutObject handles storing a key-value pair
func (h *Handler) PutObject(c *fiber.Ctx) error {
	var req models.PutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "Invalid JSON format",
		})
	}

	if req.Key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "Key field is required",
		})
	}

	collection := c.Query("collection", "default")

	if err := h.store.Put(collection, req.Key, req.Value); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.PutResponse{
		Message: "Object stored successfully",
		Key:     req.Key,
	})
}

// GetObject handles retrieving a value by key
func (h *Handler) GetObject(c *fiber.Ctx) error {
	key := c.Params("key")
	collection := c.Query("collection", "default")

	value, err := h.store.Get(collection, key)
	if err != nil {
		if err == storage.ErrKeyNotFound {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
				Error: "Key not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.GetResponse{
		Key:   key,
		Value: value,
	})
}

// ListObjects handles listing all objects in a collection
func (h *Handler) ListObjects(c *fiber.Ctx) error {
	collection := c.Query("collection", "default")

	objects, err := h.store.List(collection)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.ListResponse{
		Count:   len(objects),
		Objects: objects,
	})
}

// DeleteObject handles deleting a key-value pair
func (h *Handler) DeleteObject(c *fiber.Ctx) error {
	key := c.Params("key")
	collection := c.Query("collection", "default")

	if err := h.store.Delete(collection, key); err != nil {
		if err == storage.ErrKeyNotFound {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
				Error: "Key not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(models.DeleteResponse{
		Message: "Object deleted successfully",
		Key:     key,
	})
}
