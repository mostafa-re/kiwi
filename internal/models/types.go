package models

// PutRequest represents the request body for storing an object
type PutRequest struct {
	Key   string      `json:"key" validate:"required"`
	Value interface{} `json:"value"`
}

// PutResponse represents the response after storing an object
type PutResponse struct {
	Message string `json:"message"`
	Key     string `json:"key"`
}

// GetResponse represents the response when retrieving an object
type GetResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// ListResponse represents the response when listing all objects
type ListResponse struct {
	Count   int                    `json:"count"`
	Objects map[string]interface{} `json:"objects"`
}

// DeleteResponse represents the response after deleting an object
type DeleteResponse struct {
	Message string `json:"message"`
	Key     string `json:"key"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}
