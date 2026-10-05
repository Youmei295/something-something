package main

import (
	"github.com/youmei295/something-something/src/backend/internal/services"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// cmd/conversation owns the host inbox: listing, reading, replying, and labels.
func main() {
	services.Run("conversation", transporthttp.Enable{Conversation: true})
}
