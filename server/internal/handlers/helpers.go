package handlers

// ErrorResponse is the standard JSON error envelope used across all API responses.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody represents the code and message inside an error response.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
