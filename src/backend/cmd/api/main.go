package main

import (
	"github.com/youmei295/something-something/src/backend/internal/services"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// cmd/api runs every context in one process (modular monolith). For the
// microservice layout, run cmd/identity, cmd/submission, cmd/conversation and
// cmd/attachment behind cmd/gateway instead.
func main() {
	services.Run("api", transporthttp.Enable{
		Identity:     true,
		Submission:   true,
		Conversation: true,
		Attachment:   true,
	})
}
