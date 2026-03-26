package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/notifications"
	notificationstypes "github.com/aws/aws-sdk-go-v2/service/notifications/types"
	"github.com/aws/smithy-go"

	"gocloud-cli/internal/models"
)

const (
	defaultMaxConcurrentEventDetailRequests = 8
	debugJSONMaxBytes                       = 64 * 1024 // keep stderr usable
)

// notificationsAPI is the minimal AWS SDK surface we need.
// Keeping this tiny makes it easy to unit test without hitting AWS.
type notificationsAPI interface {
	ListManagedNotificationEvents(ctx context.Context, params *notifications.ListManagedNotificationEventsInput, optFns ...func(*notifications.Options)) (*notifications.ListManagedNotificationEventsOutput, error)
	GetManagedNotificationEvent(ctx context.Context, params *notifications.GetManagedNotificationEventInput, optFns ...func(*notifications.Options)) (*notifications.GetManagedNotificationEventOutput, error)
}

// Manager implements API using AWS User Notifications (AWS SDK v2).
// Naming matches the rest of the repo (e.g. secrets.Manager).
type Manager struct {
	config *models.Config

	loadConfigForProfile func(ctx context.Context, profileName string, region string) (aws.Config, error)
	newClient            func(cfg aws.Config) notificationsAPI

	cacheMu sync.RWMutex
	clients map[string]notificationsAPI // keyed by envKey|region
}

type eventTimes struct {
	start *time.Time
	end   *time.Time
}

func NewManager(config *models.Config) (*Manager, error) {
	if config == nil || config.Infrastructure == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.Infrastructure.Client == "" {
		return nil, fmt.Errorf("infrastructure.client is required")
	}

	m := &Manager{
		config:  config,
		clients: make(map[string]notificationsAPI),
	}
	m.loadConfigForProfile = func(ctx context.Context, profileName string, region string) (aws.Config, error) {
		configFile := os.Getenv("AWS_CONFIG_FILE")
		if configFile == "" {
			configFile = ".aws/config"
		}
		if region == "" {
			// Notifications is not available in all regions. Default to a known good region.
			region = "us-east-1"
		}
		return awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithSharedConfigProfile(profileName),
			awsconfig.WithSharedConfigFiles([]string{configFile}),
		)
	}
	m.newClient = func(cfg aws.Config) notificationsAPI {
		return notifications.NewFromConfig(cfg)
	}
	return m, nil
}

func (m *Manager) getClientForEnv(ctx context.Context, envKey string, region string) (notificationsAPI, error) {
	cacheKey := fmt.Sprintf("%s|%s", envKey, region)

	m.cacheMu.RLock()
	if cl, ok := m.clients[cacheKey]; ok {
		m.cacheMu.RUnlock()
		return cl, nil
	}
	m.cacheMu.RUnlock()

	profileName := fmt.Sprintf("%s-%s", m.config.Infrastructure.Client, envKey)
	cfg, err := m.loadConfigForProfile(ctx, profileName, region)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for profile %s: %w", profileName, err)
	}
	cl := m.newClient(cfg)

	m.cacheMu.Lock()
	m.clients[cacheKey] = cl
	m.cacheMu.Unlock()

	return cl, nil
}

func (m *Manager) ListManagedEvents(ctx context.Context, envKey string, accountID string, opts ListOptions) ([]ManagedEvent, error) {
	start := opts.StartTime
	end := opts.EndTime
	if start.IsZero() {
		start = time.Now().UTC().Add(-14 * 24 * time.Hour)
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}
	now := time.Now().UTC()

	var out []ManagedEvent
	var next *string

	// Local cache (per invocation) to avoid duplicate detail calls for the same ARN.
	// The list API can include repeated ARNs across pagination windows.
	timesCache := make(map[string]eventTimes, 128)

	// Notifications API is not enabled in all regions. Try known good regions.
	regionsToTry := []string{"us-east-1", "us-west-2"}
	var lastErr error

	for _, region := range regionsToTry {
		cl, err := m.getClientForEnv(ctx, envKey, region)
		if err != nil {
			lastErr = err
			continue
		}

		out = out[:0]
		next = nil
		for {
			resp, err := cl.ListManagedNotificationEvents(ctx, &notifications.ListManagedNotificationEventsInput{
				RelatedAccount: aws.String(accountID),
				StartTime:      &start,
				EndTime:        &end,
				MaxResults:     aws.Int32(100),
				NextToken:      next,
				Locale:         notificationstypes.LocaleCodeEnUs,
			})
			if err != nil {
				lastErr = err
				out = out[:0]
				break
			}

			if w := debugWriter(ctx); w != nil {
				// Best-effort: dump the raw response (can be large).
				if b, mErr := json.MarshalIndent(resp, "", "  "); mErr == nil {
					_, _ = fmt.Fprintf(w, "\n[notifications] ListManagedNotificationEvents env=%s region=%s account=%s\n%s\n", envKey, region, accountID, truncateDebugJSON(string(b), debugJSONMaxBytes))
				} else {
					_, _ = fmt.Fprintf(w, "\n[notifications] ListManagedNotificationEvents env=%s region=%s account=%s (marshal error: %v)\n", envKey, region, accountID, mErr)
				}
			}

			// Convert overviews to our model and apply filters that do NOT require detail calls.
			// This reduces the number of GetManagedNotificationEvent requests.
			page := make([]ManagedEvent, 0, len(resp.ManagedNotificationEvents))
			for _, ev := range resp.ManagedNotificationEvents {
				me := fromOverview(envKey, accountID, ev)

				if opts.OnlyAWSManaged && me.Source != "" && !strings.HasPrefix(me.Source, "aws.") {
					continue
				}
				if opts.OnlyActive && !isBestEffortActive(me) {
					continue
				}
				page = append(page, me)
			}

			// Enrich start/end times in parallel with local cache.
			enrichManagedEventTimes(ctx, cl, page, timesCache, debugWriter(ctx), defaultMaxConcurrentEventDetailRequests)

			for _, me := range page {
				if !opts.IncludeEnded && me.EndTime != nil && me.EndTime.Before(now) {
					continue
				}
				out = append(out, me)
			}

			if resp.NextToken == nil || *resp.NextToken == "" {
				lastErr = nil
				break
			}
			next = resp.NextToken
		}

		if lastErr == nil {
			return out, nil
		}
	}

	return nil, lastErr
}

func enrichManagedEventTimes(
	ctx context.Context,
	cl notificationsAPI,
	events []ManagedEvent,
	cache map[string]eventTimes,
	debugW io.Writer,
	maxConcurrent int,
) {
	if len(events) == 0 {
		return
	}
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrentEventDetailRequests
	}

	type job struct {
		arn string
		idx []int
	}

	// Group indexes by ARN so duplicates are fetched once.
	idxByArn := make(map[string][]int, len(events))
	for i := range events {
		arn := strings.TrimSpace(events[i].Arn)
		if arn == "" {
			continue
		}
		if t, ok := cache[arn]; ok {
			events[i].StartTime = t.start
			events[i].EndTime = t.end
			continue
		}
		idxByArn[arn] = append(idxByArn[arn], i)
	}
	if len(idxByArn) == 0 {
		return
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex // protects cache + writes back to events

	toFetch := make([]job, 0, len(idxByArn))
	for arn, idxs := range idxByArn {
		toFetch = append(toFetch, job{arn: arn, idx: idxs})
	}

	for _, j := range toFetch {
		// Respect cancellation early.
		if ctx.Err() != nil {
			return
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()

			startT, endT, derr := fetchManagedEventTimes(ctx, cl, j.arn)
			if derr != nil {
				if debugW != nil {
					_, _ = fmt.Fprintf(debugW, "[notifications] GetManagedNotificationEvent failed arn=%s err=%v\n", j.arn, derr)
				}
				// Keep cache entry as empty (prevents repeated attempts in this run).
				return
			}

			// Store in cache and write back to the event.
			mu.Lock()
			cache[j.arn] = eventTimes{start: startT, end: endT}
			for _, idx := range j.idx {
				events[idx].StartTime = startT
				events[idx].EndTime = endT
			}
			mu.Unlock()

		}(j)
	}

	wg.Wait()
}

var fetchManagedEventTimes = getManagedEventTimes

func getManagedEventTimes(ctx context.Context, cl notificationsAPI, arn string) (start *time.Time, end *time.Time, err error) {
	resp, err := cl.GetManagedNotificationEvent(ctx, &notifications.GetManagedNotificationEventInput{
		Arn:    aws.String(arn),
		Locale: notificationstypes.LocaleCodeEnUs,
	})
	if err != nil {
		return nil, nil, err
	}
	if w := debugWriter(ctx); w != nil {
		if b, mErr := json.MarshalIndent(resp, "", "  "); mErr == nil {
			_, _ = fmt.Fprintf(w, "[notifications] GetManagedNotificationEvent arn=%s\n%s\n", arn, truncateDebugJSON(string(b), debugJSONMaxBytes))
		} else {
			_, _ = fmt.Fprintf(w, "[notifications] GetManagedNotificationEvent arn=%s (marshal error: %v)\n", arn, mErr)
		}
	}
	if resp.Content == nil {
		return nil, nil, nil
	}

	start = resp.Content.StartTime
	end = resp.Content.EndTime

	// Some managed events (notably AWS Health ones) may provide start/end in message dimensions
	// instead of the top-level Content.StartTime/EndTime.
	if resp.Content.MessageComponents != nil && len(resp.Content.MessageComponents.Dimensions) > 0 {
		if start == nil {
			if t, ok := parseDimensionTime(resp.Content.MessageComponents.Dimensions, "Start time"); ok {
				start = &t
			}
		}
		if end == nil {
			if t, ok := parseDimensionTime(resp.Content.MessageComponents.Dimensions, "End time"); ok {
				end = &t
			}
		}
	}

	return start, end, nil
}

func truncateDebugJSON(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return s
	}
	if len(s) <= maxBytes {
		return s
	}
	// Keep the prefix (often contains the schema/fields) and show truncation.
	return s[:maxBytes] + "\n... (truncated) ..."
}

func parseDimensionTime(dims []notificationstypes.Dimension, key string) (time.Time, bool) {
	for _, d := range dims {
		if aws.ToString(d.Name) != key {
			continue
		}
		v := strings.TrimSpace(aws.ToString(d.Value))
		if v == "" {
			return time.Time{}, false
		}
		// Example: "Fri, 6 Feb 2026 23:00:00 GMT"
		const layout = "Mon, 2 Jan 2006 15:04:05 MST"
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func fromOverview(envKey, accountID string, ov notificationstypes.ManagedNotificationEventOverview) ManagedEvent {
	me := ManagedEvent{
		EnvKey:    envKey,
		AccountID: accountID,
		Arn:       aws.ToString(ov.Arn),
	}

	if ov.CreationTime != nil {
		me.CreationTime = *ov.CreationTime
	}
	if ov.NotificationEvent != nil {
		me.EventStatus = string(ov.NotificationEvent.EventStatus)
		me.NotificationType = string(ov.NotificationEvent.NotificationType)
		if ov.NotificationEvent.MessageComponents != nil {
			me.Headline = aws.ToString(ov.NotificationEvent.MessageComponents.Headline)
		}
		if ov.NotificationEvent.SourceEventMetadata != nil {
			me.Source = aws.ToString(ov.NotificationEvent.SourceEventMetadata.Source)
			me.EventType = aws.ToString(ov.NotificationEvent.SourceEventMetadata.EventType)
			me.OriginRegion = aws.ToString(ov.NotificationEvent.SourceEventMetadata.EventOriginRegion)
		}
	}

	return me
}

func isBestEffortActive(ev ManagedEvent) bool {
	// EventStatus in Notifications is about rule health, not "open/upcoming".
	// We approximate "active" as:
	// - UNHEALTHY, or
	// - notificationType ALERT/WARNING, and
	// - headline does not look resolved.
	if strings.EqualFold(ev.EventStatus, "UNHEALTHY") {
		return true
	}
	switch strings.ToUpper(ev.NotificationType) {
	case "ALERT", "WARNING":
		hl := strings.ToLower(ev.Headline)
		if strings.Contains(hl, "resolved") || strings.Contains(hl, "closed") {
			return false
		}
		return true
	default:
		return false
	}
}

func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AccessDeniedException"
	}
	return false
}
