package main

import (
	"github.com/youmei295/something-something/src/backend/internal/services"
	transporthttp "github.com/youmei295/something-something/src/backend/internal/transport/http"
)

// cmd/attachment owns file upload/download via presigned object-storage URLs.
func main() {
	services.Run("attachment", transporthttp.Enable{Attachment: true})
}
