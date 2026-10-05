package main

import (
	"github.com/youmei295/something-something/src/backend/internal/services"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// cmd/identity owns host authentication: login, logout, and session lookup.
func main() {
	services.Run("identity", transporthttp.Enable{Identity: true})
}
