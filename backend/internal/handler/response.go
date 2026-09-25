package handler

import (
	"github.com/gin-gonic/gin"
)

// ResponseSuccess envelopes successful API responses.
type ResponseSuccess struct {
	Data interface{} `json:"data"`
}

// ErrorDetail describes an API error.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ResponseError envelopes error API responses.
type ResponseError struct {
	Error ErrorDetail `json:"error"`
}

// RespondSuccess sends a standard JSON success response.
func RespondSuccess(c *gin.Context, status int, data interface{}) {
	c.JSON(status, ResponseSuccess{
		Data: data,
	})
}

// RespondError sends a standard JSON error response.
func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ResponseError{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
