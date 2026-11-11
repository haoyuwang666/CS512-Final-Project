// TODO(Haoyu): Implement leader-only reservation/task coordination and replay; repair is deliberately unavailable until durable transitions exist.
package control

import (
	"context"
	"cooperative.video/internal/domain"
	"errors"
)

var ErrNotImplemented = errors.New("durable task coordination not implemented")

type Coordinator interface {
	SubmitRepair(context.Context, domain.Task) error
	Recover(context.Context) error
}
