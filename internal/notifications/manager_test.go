package notifications

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdknotifications "github.com/aws/aws-sdk-go-v2/service/notifications"
	notificationstypes "github.com/aws/aws-sdk-go-v2/service/notifications/types"

	"gocloud-cli/internal/models"
)

func TestManager_CachesClientPerEnv(t *testing.T) {
	cfg := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client: "test",
			Region: "us-east-1",
		},
	}
	c, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	var calls int32
	c.loadConfigForProfile = func(_ context.Context, _ string, _ string) (aws.Config, error) {
		atomic.AddInt32(&calls, 1)
		return aws.Config{Region: "us-east-1"}, nil
	}

	_, err = c.getClientForEnv(context.Background(), "prd", "us-east-1")
	if err != nil {
		t.Fatalf("getClientForEnv first: %v", err)
	}
	_, err = c.getClientForEnv(context.Background(), "prd", "us-east-1")
	if err != nil {
		t.Fatalf("getClientForEnv second: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected loadConfigForProfile to be called once, got %d", got)
	}
}

func TestParseDimensionTime(t *testing.T) {
	dims := []notificationstypes.Dimension{
		{Name: aws.String("Start time"), Value: aws.String("Tue, 17 Mar 2026 05:00:00 GMT")},
		{Name: aws.String("End time"), Value: aws.String("Tue, 24 Mar 2026 05:00:00 GMT")},
	}

	start, ok := parseDimensionTime(dims, "Start time")
	if !ok {
		t.Fatal("expected start time to parse")
	}
	if got := start.Format("2006-01-02T15:04:05Z"); got != "2026-03-17T05:00:00Z" {
		t.Fatalf("unexpected start time: %s", got)
	}

	end, ok := parseDimensionTime(dims, "End time")
	if !ok {
		t.Fatal("expected end time to parse")
	}
	if got := end.Format("2006-01-02T15:04:05Z"); got != "2026-03-24T05:00:00Z" {
		t.Fatalf("unexpected end time: %s", got)
	}
}

// This test focuses on the local per-run ARN cache + deduplication behavior.
// We don't hit AWS; instead we validate that duplicates don't cause duplicate fetches.
func TestEnrichManagedEventTimes_DedupesByArn(t *testing.T) {
	ctx := context.Background()

	events := []ManagedEvent{
		{Arn: "arn:1"},
		{Arn: "arn:1"},
		{Arn: "arn:2"},
		{Arn: ""},
	}
	cache := map[string]eventTimes{}

	var calls int32

	start1 := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	start2 := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
	end2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

	orig := fetchManagedEventTimes
	fetchManagedEventTimes = func(_ context.Context, _ notificationsAPI, arn string) (*time.Time, *time.Time, error) {
		atomic.AddInt32(&calls, 1)
		switch arn {
		case "arn:1":
			return &start1, &end1, nil
		case "arn:2":
			return &start2, &end2, nil
		default:
			return nil, nil, nil
		}
	}
	defer func() { fetchManagedEventTimes = orig }()

	enrichManagedEventTimes(ctx, nil, events, cache, nil, 4)

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 fetch calls (unique arns), got %d", got)
	}

	if events[0].StartTime == nil || events[1].StartTime == nil || events[2].StartTime == nil {
		t.Fatalf("expected start times to be populated")
	}
	if events[0].StartTime.Format(time.RFC3339) != start1.Format(time.RFC3339) {
		t.Fatalf("unexpected start time for arn:1: %v", events[0].StartTime)
	}
	if events[1].EndTime.Format(time.RFC3339) != end1.Format(time.RFC3339) {
		t.Fatalf("unexpected end time for arn:1: %v", events[1].EndTime)
	}
	if events[2].StartTime.Format(time.RFC3339) != start2.Format(time.RFC3339) {
		t.Fatalf("unexpected start time for arn:2: %v", events[2].StartTime)
	}
}

type fakeSDKNotifications struct {
	listCalls int32
	getCalls  int32

	// list responses keyed by nextToken ("" for first page)
	listByToken map[string]*sdknotifications.ListManagedNotificationEventsOutput
	listErr     error

	getByArn map[string]*sdknotifications.GetManagedNotificationEventOutput
	getErr   error
}

func (f *fakeSDKNotifications) ListManagedNotificationEvents(_ context.Context, in *sdknotifications.ListManagedNotificationEventsInput, _ ...func(*sdknotifications.Options)) (*sdknotifications.ListManagedNotificationEventsOutput, error) {
	atomic.AddInt32(&f.listCalls, 1)
	if f.listErr != nil {
		return nil, f.listErr
	}
	token := ""
	if in.NextToken != nil {
		token = aws.ToString(in.NextToken)
	}
	if out, ok := f.listByToken[token]; ok {
		return out, nil
	}
	return &sdknotifications.ListManagedNotificationEventsOutput{}, nil
}

func (f *fakeSDKNotifications) GetManagedNotificationEvent(_ context.Context, in *sdknotifications.GetManagedNotificationEventInput, _ ...func(*sdknotifications.Options)) (*sdknotifications.GetManagedNotificationEventOutput, error) {
	atomic.AddInt32(&f.getCalls, 1)
	if f.getErr != nil {
		return nil, f.getErr
	}
	arn := aws.ToString(in.Arn)
	if out, ok := f.getByArn[arn]; ok {
		return out, nil
	}
	return &sdknotifications.GetManagedNotificationEventOutput{}, nil
}

func TestManager_ListManagedEvents_Paginates_Dedupes_AndFiltersEnded(t *testing.T) {
	cfg := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client: "test",
			Region: "us-east-1",
		},
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	m.loadConfigForProfile = func(_ context.Context, _ string, region string) (aws.Config, error) {
		return aws.Config{Region: region}, nil
	}

	now := time.Now().UTC()
	pastEnd := now.Add(-1 * time.Hour)
	futureEnd := now.Add(24 * time.Hour)

	arn1 := "arn:1" // appears in both pages (dedupe)
	arn2 := "arn:2" // ended in past (filtered)

	fake := &fakeSDKNotifications{
		listByToken: map[string]*sdknotifications.ListManagedNotificationEventsOutput{
			"": {
				ManagedNotificationEvents: []notificationstypes.ManagedNotificationEventOverview{
					{
						Arn:          aws.String(arn1),
						CreationTime: aws.Time(now),
						NotificationEvent: &notificationstypes.ManagedNotificationEventSummary{
							NotificationType: notificationstypes.NotificationTypeAnnouncement,
							MessageComponents: &notificationstypes.MessageComponentsSummary{
								Headline: aws.String("headline 1"),
							},
							SourceEventMetadata: &notificationstypes.ManagedSourceEventMetadataSummary{
								Source:            aws.String("aws.health"),
								EventType:         aws.String("AWS Health Event"),
								EventOriginRegion: aws.String("us-east-1"),
							},
							EventStatus: notificationstypes.EventStatusHealthy,
						},
					},
				},
				NextToken: aws.String("t2"),
			},
			"t2": {
				ManagedNotificationEvents: []notificationstypes.ManagedNotificationEventOverview{
					{
						Arn:          aws.String(arn1), // duplicate
						CreationTime: aws.Time(now),
						NotificationEvent: &notificationstypes.ManagedNotificationEventSummary{
							NotificationType: notificationstypes.NotificationTypeAnnouncement,
							MessageComponents: &notificationstypes.MessageComponentsSummary{
								Headline: aws.String("headline 1 dup"),
							},
							SourceEventMetadata: &notificationstypes.ManagedSourceEventMetadataSummary{
								Source:    aws.String("aws.health"),
								EventType: aws.String("AWS Health Event"),
							},
							EventStatus: notificationstypes.EventStatusHealthy,
						},
					},
					{
						Arn:          aws.String(arn2),
						CreationTime: aws.Time(now),
						NotificationEvent: &notificationstypes.ManagedNotificationEventSummary{
							NotificationType: notificationstypes.NotificationTypeAnnouncement,
							MessageComponents: &notificationstypes.MessageComponentsSummary{
								Headline: aws.String("ended event"),
							},
							SourceEventMetadata: &notificationstypes.ManagedSourceEventMetadataSummary{
								Source:    aws.String("aws.health"),
								EventType: aws.String("AWS Health Event"),
							},
							EventStatus: notificationstypes.EventStatusHealthy,
						},
					},
				},
			},
		},
		getByArn: map[string]*sdknotifications.GetManagedNotificationEventOutput{
			arn1: {Content: &notificationstypes.ManagedNotificationEvent{StartTime: aws.Time(now), EndTime: aws.Time(futureEnd)}},
			arn2: {Content: &notificationstypes.ManagedNotificationEvent{StartTime: aws.Time(now), EndTime: aws.Time(pastEnd)}},
		},
	}
	m.newClient = func(_ aws.Config) notificationsAPI { return fake }

	out, err := m.ListManagedEvents(context.Background(), "prd", "123456789012", ListOptions{
		StartTime:      now.Add(-24 * time.Hour),
		EndTime:        now,
		OnlyAWSManaged: true,
		OnlyActive:     false,
	})
	if err != nil {
		t.Fatalf("ListManagedEvents: %v", err)
	}

	// arn2 must be filtered (ended).
	for _, e := range out {
		if e.Arn == arn2 {
			t.Fatalf("expected ended event %s to be filtered out", arn2)
		}
	}
	if got := atomic.LoadInt32(&fake.getCalls); got != 2 {
		t.Fatalf("expected GetManagedNotificationEvent called for 2 unique arns, got %d", got)
	}
}

func TestManager_ListManagedEvents_IncludesEndedWhenIncludeEnded(t *testing.T) {
	cfg := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client: "test",
			Region: "us-east-1",
		},
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	m.loadConfigForProfile = func(_ context.Context, _ string, region string) (aws.Config, error) {
		return aws.Config{Region: region}, nil
	}

	now := time.Now().UTC()
	pastEnd := now.Add(-1 * time.Hour)
	arnEnded := "arn:ended"

	fake := &fakeSDKNotifications{
		listByToken: map[string]*sdknotifications.ListManagedNotificationEventsOutput{
			"": {
				ManagedNotificationEvents: []notificationstypes.ManagedNotificationEventOverview{
					{
						Arn:          aws.String(arnEnded),
						CreationTime: aws.Time(now),
						NotificationEvent: &notificationstypes.ManagedNotificationEventSummary{
							NotificationType: notificationstypes.NotificationTypeAnnouncement,
							MessageComponents: &notificationstypes.MessageComponentsSummary{
								Headline: aws.String("past window"),
							},
							SourceEventMetadata: &notificationstypes.ManagedSourceEventMetadataSummary{
								Source:            aws.String("aws.health"),
								EventType:         aws.String("AWS Health Event"),
								EventOriginRegion: aws.String("us-east-1"),
							},
							EventStatus: notificationstypes.EventStatusHealthy,
						},
					},
				},
			},
		},
		getByArn: map[string]*sdknotifications.GetManagedNotificationEventOutput{
			arnEnded: {Content: &notificationstypes.ManagedNotificationEvent{StartTime: aws.Time(now.Add(-48 * time.Hour)), EndTime: aws.Time(pastEnd)}},
		},
	}
	m.newClient = func(_ aws.Config) notificationsAPI { return fake }

	out, err := m.ListManagedEvents(context.Background(), "prd", "123456789012", ListOptions{
		StartTime:      now.Add(-24 * time.Hour),
		EndTime:        now,
		OnlyAWSManaged: true,
		OnlyActive:     false,
		IncludeEnded:   true,
	})
	if err != nil {
		t.Fatalf("ListManagedEvents: %v", err)
	}
	if len(out) != 1 || out[0].Arn != arnEnded {
		t.Fatalf("expected one ended event included, got %+v", out)
	}
}

func TestManager_DebugJSON_Truncates(t *testing.T) {
	cfg := &models.Config{Infrastructure: &models.InfrastructureConfig{Client: "test", Region: "us-east-1"}}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	m.loadConfigForProfile = func(_ context.Context, _ string, region string) (aws.Config, error) {
		return aws.Config{Region: region}, nil
	}

	huge := strings.Repeat("x", debugJSONMaxBytes+1024)
	now := time.Now().UTC()
	fake := &fakeSDKNotifications{
		listByToken: map[string]*sdknotifications.ListManagedNotificationEventsOutput{
			"": {
				ManagedNotificationEvents: []notificationstypes.ManagedNotificationEventOverview{
					{Arn: aws.String("arn:1"), CreationTime: aws.Time(now)},
				},
			},
		},
		getByArn: map[string]*sdknotifications.GetManagedNotificationEventOutput{
			"arn:1": {
				Content: &notificationstypes.ManagedNotificationEvent{
					// Force a big JSON dump via MessageComponents-like payload
					MessageComponents: &notificationstypes.MessageComponents{
						Headline: aws.String(huge),
					},
					TextParts: map[string]notificationstypes.TextPartValue{
						"x": {Type: notificationstypes.TextPartTypeLocalizedText, TextByLocale: map[string]string{"en_US": huge}},
					},
				},
			},
		},
	}
	m.newClient = func(_ aws.Config) notificationsAPI { return fake }

	var b strings.Builder
	ctx := WithDebugJSON(context.Background(), &b)
	_, err = m.ListManagedEvents(ctx, "prd", "123456789012", ListOptions{
		StartTime:      now.Add(-24 * time.Hour),
		EndTime:        now,
		OnlyAWSManaged: false,
		OnlyActive:     false,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("ListManagedEvents: %v", err)
	}
	if !strings.Contains(b.String(), "(truncated)") {
		t.Fatalf("expected debug output to be truncated")
	}
}
