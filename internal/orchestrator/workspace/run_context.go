package workspace

import (
	"context"

	"github.com/erisristemena/relay/internal/storage/sqlite"
)

type runExecutionContext struct {
	SessionID string
	RunID     string
	Emit      func(StreamEnvelope) error
	Role      sqlite.AgentRole
	Model     string
}

type runExecutionContextKey struct{}

func withRunExecutionContext(ctx context.Context, value runExecutionContext) context.Context {
	return context.WithValue(ctx, runExecutionContextKey{}, value)
}

func runExecutionContextFromContext(ctx context.Context) (runExecutionContext, bool) {
	value, ok := ctx.Value(runExecutionContextKey{}).(runExecutionContext)
	return value, ok
}

type streamSubscriberKey struct{}

func WithStreamSubscriber(ctx context.Context, subscriberID string) context.Context {
	return context.WithValue(ctx, streamSubscriberKey{}, subscriberID)
}

func streamSubscriberFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(streamSubscriberKey{}).(string)
	return value, ok
}
