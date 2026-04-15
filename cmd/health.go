package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	cfg "gocloud-cli/internal/config"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/notifications"
	"gocloud-cli/internal/utils"
)

var (
	healthConfigFile   string
	healthCheckAll     bool
	healthCheckEnv     string
	healthManagedDays  int
	healthOutputFormat string
	healthDebugJSON    bool
	healthIncludeEnded bool
)

var newNotificationsAPI = func(config *models.Config) (notifications.API, error) {
	return notifications.NewManager(config)
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check AWS managed health-related notifications per environment",
	Long:  "Check AWS User Notifications / Notification Center (AWS managed) per environment/account.",
}

var healthCheckCmd = &cobra.Command{
	Use:           "check",
	Short:         "List managed notification events per environment",
	Long:          "Lists AWS managed notification events via AWS User Notifications / Notification Center, grouped by environment.",
	RunE:          runHealthCheck,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	healthCmd.AddCommand(healthCheckCmd)

	healthCmd.PersistentFlags().StringVarP(&healthConfigFile, "config", "c", "gocloud.yaml", "Configuration file path")
	healthCheckCmd.Flags().BoolVar(&healthCheckAll, "all", false, "Check all environments in the project")
	healthCheckCmd.Flags().StringVar(&healthCheckEnv, "environment", "", "Check a specific environment key")
	healthCheckCmd.Flags().IntVar(&healthManagedDays, "managed-days", 90, "When falling back to Notification Center, look back this many days for managed events")
	healthCheckCmd.Flags().BoolVar(&healthIncludeEnded, "include-ended", false, "Include managed events whose end time is already in the past")
	healthCheckCmd.Flags().StringVar(&healthOutputFormat, "output", "list", "Output format: list or table")
	healthCheckCmd.Flags().BoolVar(&healthDebugJSON, "debug-json", false, "Print raw JSON payloads from AWS Notifications API (for debugging)")
}

func runHealthCheck(cmd *cobra.Command, _ []string) error {
	// Validate selection flags first (keeps behavior consistent and avoids config dependency).
	if !healthCheckAll && healthCheckEnv == "" {
		fmt.Println("Must specify --environment or --all")
		return nil
	}

	config, err := cfg.LoadConfigWithPathAndAWS(healthConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	envKeys := selectEnvKeysForHealth(config)
	if len(envKeys) == 0 {
		fmt.Println("Must specify --environment or --all")
		return nil
	}

	notifAPI, err := newNotificationsAPI(config)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS Notifications client: %w", err)
	}

	utils.PrintInfo("🔍 Checking managed notification events...")
	fmt.Println()

	ctx := context.Background()
	if healthDebugJSON {
		ctx = notifications.WithDebugJSON(ctx, os.Stderr)
	}

	var rows []healthRow
	printedCount := 0
	noAccessCount := 0
	notificationsAccessDeniedCount := 0
	notificationsOtherErrorCount := 0
	envWithEvents := map[string]bool{}

	outputMode := strings.ToLower(strings.TrimSpace(healthOutputFormat))
	if outputMode == "" {
		outputMode = "list"
	}

	// In list mode, stream per-environment output as we fetch.
	streamList := outputMode != "table"

	for _, envKey := range envKeys {
		accountID, ok := accountIDForHealthEnv(config.Infrastructure, envKey)
		if !ok {
			if envKey == "org" {
				utils.PrintWarning("⚠️  %s: SKIPPED (organization layer not enabled in config)", envKey)
			} else if envKey == "sec" {
				utils.PrintWarning("⚠️  %s: SKIPPED (security layer not enabled in config)", envKey)
			} else {
				utils.PrintWarning("⚠️  %s: SKIPPED (environment not found in config)", envKey)
			}
			continue
		}

		var envRows []healthRow
		used, nAccessDenied, nCred, nOther := fetchNotificationsRows(ctx, notifAPI, envKey, accountID, &envRows, &envWithEvents)
		notificationsAccessDeniedCount += nAccessDenied
		noAccessCount += nCred
		notificationsOtherErrorCount += nOther
		if used && streamList {
			// Print incrementally per environment.
			printHealthListEnv(os.Stdout, envKey, envRows)
			printedCount += len(envRows)
		} else if used {
			rows = append(rows, envRows...)
			printedCount += len(envRows)
		}
	}

	if printedCount == 0 && outputMode != "table" {
		// In streaming list mode we don't keep rows in memory.
		utils.PrintSuccess("✅ No managed events found (for the selected environments).")
		if noAccessCount > 0 {
			utils.PrintWarning("⚠️  %d environments were skipped due to NO_ACCESS", noAccessCount)
		}
		if notificationsAccessDeniedCount > 0 {
			utils.PrintWarning("⚠️  %d environments had Notification Center access denied", notificationsAccessDeniedCount)
		}
		if notificationsOtherErrorCount > 0 {
			utils.PrintWarning("⚠️  %d environments had Notification Center errors", notificationsOtherErrorCount)
		}
		return nil
	}

	if outputMode == "table" && len(rows) == 0 {
		utils.PrintSuccess("✅ No managed events found (for the selected environments).")
		if noAccessCount > 0 {
			utils.PrintWarning("⚠️  %d environments were skipped due to NO_ACCESS", noAccessCount)
		}
		if notificationsAccessDeniedCount > 0 {
			utils.PrintWarning("⚠️  %d environments had Notification Center access denied", notificationsAccessDeniedCount)
		}
		if notificationsOtherErrorCount > 0 {
			utils.PrintWarning("⚠️  %d environments had Notification Center errors", notificationsOtherErrorCount)
		}
		return nil
	}

	if outputMode == "table" {
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Env != rows[j].Env {
				return rows[i].Env < rows[j].Env
			}
			if rows[i].Source != rows[j].Source {
				return rows[i].Source < rows[j].Source
			}
			if rows[i].Category != rows[j].Category {
				return rows[i].Category < rows[j].Category
			}
			return rows[i].Start < rows[j].Start
		})

		printHealthTable(os.Stdout, rows)
	}

	fmt.Println()
	utils.PrintInfo("Summary: %d events across %d environments", printedCount, len(envWithEvents))
	if noAccessCount > 0 {
		utils.PrintWarning("⚠️  %d environments had credential/access issues", noAccessCount)
	}

	return nil
}

func selectEnvKeysForHealth(config *models.Config) []string {
	if config == nil || config.Infrastructure == nil {
		return nil
	}
	if healthCheckAll {
		keys := make([]string, 0, len(config.Infrastructure.Environments))
		for k := range config.Infrastructure.Environments {
			keys = append(keys, k)
		}
		if models.IsOrganizationEnabled(config.Infrastructure) {
			keys = append(keys, "org")
		}
		if models.IsSecurityEnabled(config.Infrastructure) {
			keys = append(keys, "sec")
		}
		sort.Strings(keys)
		return keys
	}
	if healthCheckEnv != "" {
		if healthCheckEnv == "org" && !models.IsOrganizationEnabled(config.Infrastructure) {
			return nil
		}
		if healthCheckEnv == "sec" && !models.IsSecurityEnabled(config.Infrastructure) {
			return nil
		}
		return []string{healthCheckEnv}
	}
	return nil
}

func accountIDForHealthEnv(infra *models.InfrastructureConfig, envKey string) (string, bool) {
	if infra == nil {
		return "", false
	}
	if envKey == "org" {
		if !models.IsOrganizationEnabled(infra) {
			return "", false
		}
		return infra.Organization.AWSAccount, true
	}
	if envKey == "sec" {
		if !models.IsSecurityEnabled(infra) {
			return "", false
		}
		return infra.Security.AWSAccount, true
	}
	envCfg, ok := infra.Environments[envKey]
	if !ok {
		return "", false
	}
	if strings.TrimSpace(envCfg.AWSAccount) == "" {
		return "", false
	}
	return envCfg.AWSAccount, true
}

type healthRow struct {
	Env      string
	Source   string
	Category string
	Status   string
	Service  string
	Region   string
	Start    string
	EndETA   string
	Title    string
}

func printHealthTable(w io.Writer, rows []healthRow) {
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ENV\tSOURCE\tCATEGORY\tSTATUS\tSERVICE\tREGION\tSTART\tEND\tTITLE")
	for _, r := range rows {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Env, r.Source, r.Category, r.Status, r.Service, r.Region, r.Start, r.EndETA, sanitizeSingleLine(r.Title))
	}
	_ = tw.Flush()
}

// isActionRequiredHeadline detects headlines that call for user action (e.g. "[Action Required] …").
func isActionRequiredHeadline(title string) bool {
	h := strings.ToLower(strings.TrimSpace(title))
	return h != "" && strings.Contains(h, "action required")
}

func healthListGlyph(title string) string {
	if isActionRequiredHeadline(title) {
		return "⚠️"
	}
	return "-"
}

func printHealthListEnv(w io.Writer, envKey string, rows []healthRow) {
	if len(rows) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "%s\n", strings.ToUpper(envKey))
	for _, r := range rows {
		summary := fmt.Sprintf("%s | %s | %s | %s | %s", r.Source, r.Category, r.Status, r.Service, r.Region)
		if r.Start != "" {
			summary = fmt.Sprintf("%s | start=%s", summary, r.Start)
		}
		if r.EndETA != "" {
			summary = fmt.Sprintf("%s | end=%s", summary, r.EndETA)
		}
		glyph := healthListGlyph(r.Title)
		_, _ = fmt.Fprintf(w, "%s %s\n", glyph, summary)
		_, _ = fmt.Fprintf(w, "  %s\n", sanitizeSingleLine(r.Title))
	}
	_, _ = fmt.Fprintln(w)
}

func fetchNotificationsRows(
	ctx context.Context,
	api notifications.API,
	envKey string,
	accountID string,
	rows *[]healthRow,
	envWithEvents *map[string]bool,
) (used bool, accessDeniedCount int, credentialCount int, otherErrCount int) {
	lookbackDays := healthManagedDays
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	end := time.Now().UTC()
	start := end.Add(time.Duration(-lookbackDays) * 24 * time.Hour)
	eventsActive, err := api.ListManagedEvents(ctx, envKey, accountID, notifications.ListOptions{
		StartTime:      start,
		EndTime:        end,
		OnlyActive:     false,
		OnlyAWSManaged: true,
		IncludeEnded:   healthIncludeEnded,
	})
	if err != nil {
		if notifications.IsAccessDenied(err) {
			accessDeniedCount++
			// Don't mark as "skipped" in the default output; keep it concise but actionable.
			utils.PrintWarning("⚠️  %s: Notification Center access denied", envKey)
			return false, accessDeniedCount, credentialCount, otherErrCount
		}
		if utils.IsCredentialError(err) {
			credentialCount++
			utils.PrintWarning("⚠️  %s: Notification Center credential error", envKey)
			return false, accessDeniedCount, credentialCount, otherErrCount
		}
		otherErrCount++
		utils.PrintWarning("⚠️  %s: Notification Center error: %v", envKey, err)
		return false, accessDeniedCount, credentialCount, otherErrCount
	}

	if len(eventsActive) == 0 {
		return false, accessDeniedCount, credentialCount, otherErrCount
	}
	(*envWithEvents)[envKey] = true
	for _, e := range eventsActive {
		if shouldExcludeHeadline(e.Headline) {
			continue
		}
		service := e.Source
		if service == "" {
			service = "notifications"
		}
		region := e.OriginRegion
		startT := e.CreationTime
		if e.StartTime != nil && !e.StartTime.IsZero() {
			startT = *e.StartTime
		}
		endETA := ""
		if e.EndTime != nil && !e.EndTime.IsZero() {
			endETA = formatTime(*e.EndTime)
		}
		(*rows) = append((*rows), healthRow{
			Env:      envKey,
			Source:   "notifications",
			Category: e.EventType,
			Status:   e.NotificationType,
			Service:  service,
			Region:   region,
			Start:    formatTime(startT),
			EndETA:   endETA,
			Title:    e.Headline,
		})
	}
	return true, accessDeniedCount, credentialCount, otherErrCount
}

func shouldExcludeHeadline(headline string) bool {
	h := strings.ToLower(strings.TrimSpace(headline))
	if h == "" {
		return false
	}

	excludeSubstrings := []string{
		"dkim setup success",
		"your certificate is renewed",
		"domain verification success",
	}
	for _, sub := range excludeSubstrings {
		if strings.Contains(h, sub) {
			return true
		}
	}
	return false
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func sanitizeSingleLine(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	return s
}
