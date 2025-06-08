package middleware

import (
	"net/http"
	"strings"
)

type Context interface {
	Request() *http.Request
	JSON(code int, obj interface{}) error
}

type HandlerFunc func(c Context) error
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type ErrorResponse struct {
	Error string `json:"error"`
}

func Auth(token string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer "+token) {
				return c.JSON(401, ErrorResponse{Error: "Unauthorized"})
			}
			return next(c)
		}
	}
}