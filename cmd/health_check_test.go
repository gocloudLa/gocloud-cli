package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gocloud-cli/internal/models"
	"gocloud-cli/internal/notifications"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = r.Close()
	return buf.String()
}

func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()
	content := `
infrastructure:
  client: "test"
  company: "gcl"
  region: "us-east-1"
  aws_sso:
    region: "us-east-1"
    start_url: "https://example.awsapps.com/start#/"
    role_name: "Admin"
  organization:
    aws_account: "999999999999"
  environments:
    dev:
      name: "Development"
      aws_account: "111111111111"
      projects: []
      workloads: []
    prd:
      name: "Production"
      aws_account: "222222222222"
      projects: []
      workloads: []
`
	path := filepath.Join(dir, "gocloud.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestHealthCheck_NoFlags(t *testing.T) {
	// Ensure globals are reset
	healthCheckAll = false
	healthCheckEnv = ""
	healthConfigFile = "gocloud.yaml"

	out := captureStdout(t, func() {
		_ = runHealthCheck(nil, nil)
	})
	if !strings.Contains(out, "Must specify --environment or --all") {
		t.Fatalf("expected missing flags message, got: %s", out)
	}
}

func TestHealthCheck_TableAndScheduledDetails(t *testing.T) {
	tmp := t.TempDir()
	_ = writeTestConfig(t, tmp)

	origCwd, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origCwd) }()

	origNotif := newNotificationsAPI
	newNotificationsAPI = func(_ *models.Config) (notifications.API, error) {
		futureEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		start := time.Date(2026, 3, 3, 1, 0, 0, 0, time.UTC)
		return &fakeNotificationsAPI{
			eventsByEnv: map[string][]notifications.ManagedEvent{
				"prd": {
					{
						EnvKey:           "prd",
						AccountID:        "222222222222",
						Arn:              "arn:aws:notifications:us-east-1:222222222222:managed-notification-event/1",
						CreationTime:     time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
						StartTime:        &start,
						EndTime:          &futureEnd,
						Headline:         "RDS scheduled maintenance window approaching",
						Source:           "aws.health",
						EventType:        "AWS Health Event",
						OriginRegion:     "us-east-1",
						NotificationType: "ALERT",
						EventStatus:      "UNHEALTHY",
					},
					{
						EnvKey:           "prd",
						AccountID:        "222222222222",
						Arn:              "arn:aws:notifications:us-east-1:222222222222:managed-notification-event/2",
						CreationTime:     time.Date(2026, 3, 25, 11, 0, 0, 0, time.UTC),
						StartTime:        &start,
						EndTime:          &futureEnd,
						Headline:         "[Action Required] Upgrade to new version of ECR Basic Scanning",
						Source:           "aws.health",
						EventType:        "AWS Health Event",
						OriginRegion:     "us-east-1",
						NotificationType: "ANNOUNCEMENT",
						EventStatus:      "HEALTHY",
					},
				},
			},
		}, nil
	}
	defer func() { newNotificationsAPI = origNotif }()

	healthCheckAll = false
	healthCheckEnv = "prd"
	healthConfigFile = "gocloud.yaml"
	healthOutputFormat = "list"

	out := captureStdout(t, func() {
		if err := runHealthCheck(nil, nil); err != nil {
			t.Fatalf("runHealthCheck error: %v", err)
		}
	})

	if !strings.Contains(out, "PRD") || !strings.Contains(out, "notifications") || !strings.Contains(out, "RDS scheduled maintenance") {
		t.Fatalf("expected list output grouped by env, got: %s", out)
	}
	if !strings.Contains(out, "- notifications") || !strings.Contains(out, "⚠️") {
		t.Fatalf("expected hyphen bullets for normal rows and ⚠️ for action-required, got: %s", out)
	}
}

func TestIsActionRequiredHeadline(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title string
		want  bool
	}{
		{"[Action Required] Upgrade ECR", true},
		{"action required: renew certificate", true},
		{"ACTION REQUIRED — maintenance", true},
		{"RDS scheduled maintenance", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isActionRequiredHeadline(tc.title); got != tc.want {
			t.Fatalf("isActionRequiredHeadline(%q) = %v, want %v", tc.title, got, tc.want)
		}
	}
}

func TestHealthCheck_OrganizationEnv(t *testing.T) {
	tmp := t.TempDir()
	_ = writeTestConfig(t, tmp)

	origCwd, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origCwd) }()

	origNotif := newNotificationsAPI
	newNotificationsAPI = func(_ *models.Config) (notifications.API, error) {
		futureEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		return &fakeNotificationsAPI{
			eventsByEnv: map[string][]notifications.ManagedEvent{
				"org": {
					{
						EnvKey:           "org",
						AccountID:        "999999999999",
						Arn:              "arn:aws:notifications:us-east-1:999999999999:managed-notification-event/1",
						CreationTime:     time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
						EndTime:          &futureEnd,
						Headline:         "Organization event",
						Source:           "aws.health",
						EventType:        "AWS Health Event",
						OriginRegion:     "us-east-1",
						NotificationType: "ANNOUNCEMENT",
						EventStatus:      "HEALTHY",
					},
				},
			},
		}, nil
	}
	defer func() { newNotificationsAPI = origNotif }()

	healthCheckAll = false
	healthCheckEnv = "org"
	healthConfigFile = "gocloud.yaml"
	healthOutputFormat = "list"

	out := captureStdout(t, func() {
		if err := runHealthCheck(nil, nil); err != nil {
			t.Fatalf("runHealthCheck error: %v", err)
		}
	})

	if !strings.Contains(out, "ORG") || !strings.Contains(out, "Organization event") {
		t.Fatalf("expected org output, got: %s", out)
	}
}

type fakeNotificationsAPI struct {
	eventsByEnv map[string][]notifications.ManagedEvent
}

func (f *fakeNotificationsAPI) ListManagedEvents(_ context.Context, envKey string, _ string, _ notifications.ListOptions) ([]notifications.ManagedEvent, error) {
	return f.eventsByEnv[envKey], nil
}

// (Removed Health API fallback tests; Health API support was removed.)
