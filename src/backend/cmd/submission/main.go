package main

import (
	"github.com/youmei295/something-something/src/backend/internal/services"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// cmd/submission accepts visitor messages and serves the tokenized visitor view.
func main() {
	services.Run("submission", transporthttp.Enable{Submission: true})
}
