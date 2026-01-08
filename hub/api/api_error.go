package api

// APIError is a lightweight error type that carries an HTTP status code.
// It is used by shared handler logic (HTTP + WebSocket).
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}
