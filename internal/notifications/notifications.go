package notifications

import (
	"context"
	"io"
	"time"
)

// ManagedEvent is a simplified view of a managed notification event from AWS User Notifications.
type ManagedEvent struct {
	EnvKey    string
	AccountID string

	Arn          string
	CreationTime time.Time
	StartTime    *time.Time
	EndTime      *time.Time

	// High-signal fields for a table view
	Headline     string
	Source       string // e.g. aws.health, aws.rds, aws.ec2 (from sourceEventMetadata.source)
	EventType    string // from sourceEventMetadata.eventType
	OriginRegion string // from sourceEventMetadata.eventOriginRegion

	NotificationType string // ALERT/WARNING/ANNOUNCEMENT/INFORMATIONAL
	EventStatus      string // HEALTHY/UNHEALTHY
}

// ListOptions controls listing and filtering.
type ListOptions struct {
	StartTime time.Time
	EndTime   time.Time

	// OnlyActive attempts to filter to "active" events.
	// The Notifications API does not expose OPEN/UPCOMING the same way Health does,
	// so this is a best-effort filter based on type/status/headline + time window.
	OnlyActive bool

	// OnlyAWSManaged includes only events originating from AWS-managed sources (aws.*).
	OnlyAWSManaged bool

	// IncludeEnded keeps events whose EndTime is before "now". By default those are omitted.
	IncludeEnded bool
}

type API interface {
	ListManagedEvents(ctx context.Context, envKey string, accountID string, opts ListOptions) ([]ManagedEvent, error)
}

type debugJSONCtxKey struct{}

type debugJSONCfg struct {
	w io.Writer
}

func WithDebugJSON(ctx context.Context, w io.Writer) context.Context {
	if w == nil {
		return ctx
	}
	return context.WithValue(ctx, debugJSONCtxKey{}, debugJSONCfg{w: w})
}

func debugWriter(ctx context.Context) io.Writer {
	v := ctx.Value(debugJSONCtxKey{})
	cfg, ok := v.(debugJSONCfg)
	if !ok {
		return nil
	}
	return cfg.w
}
