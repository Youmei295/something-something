// Package clock provides a real-time ports.Clock implementation.
package clock

import (
	"time"

	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Clock implements ports.Clock using the system clock.
type Clock struct{}

func New() Clock { return Clock{} }

func (Clock) Now() time.Time { return time.Now().UTC() }

var _ ports.Clock = Clock{}
