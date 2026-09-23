package moddeps

import (
	"context"
	"strings"
	"testing"
)

func TestUpstreamCommitLines(t *testing.T) {
	const owner, repo = "gocloudLa", "terraform-aws-wrapper-acm"
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "squash suffix",
			in:   "feat(module): add import & self-signed certificate support (#12)",
			want: []string{"gocloudLa/terraform-aws-wrapper-acm#12"},
		},
		{
			name: "multiple refs",
			in:   "fix: close gaps (#1) (#2)",
			want: []string{"gocloudLa/terraform-aws-wrapper-acm#1", "gocloudLa/terraform-aws-wrapper-acm#2"},
		},
		{
			name: "already qualified stays in subject",
			in:   "chore: follow other/repo#4",
			want: []string{"chore: follow other/repo#4"},
		},
		{
			name: "no ref",
			in:   "feat: no pull request id",
			want: []string{"feat: no pull request id"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := upstreamCommitLines(tc.in, owner, repo)
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}

	if got := upstreamCommitLines("feat: thing (#12)", "", repo); len(got) != 1 || got[0] != "feat: thing (#12)" {
		t.Fatalf("missing owner should leave subject unchanged, got %q", got)
	}
}

func TestBuildPRMetaListsUpstreamPRRefs(t *testing.T) {
	var c *Client
	meta := c.BuildPRMeta(
		context.Background(),
		"gocloudLa/terraform-aws-wrapper-acm/aws",
		"1.0.0",
		"1.1.0",
		[]string{"modules/base/main.tf"},
		[]string{
			"feat(module): add import & self-signed certificate support (#12)",
			"fix: another change (#12)",
		},
		"gocloudLa",
		"terraform-aws-wrapper-acm",
	)
	want := "- gocloudLa/terraform-aws-wrapper-acm#12"
	if strings.Count(meta.Body, want) != 1 {
		t.Fatalf("pr body should list the upstream ref once:\n%s", meta.Body)
	}
	if strings.Contains(meta.Body, "feat(module)") {
		t.Fatalf("pr body should not repeat the commit subject:\n%s", meta.Body)
	}
	if strings.Contains(meta.Title, "#12") {
		t.Fatalf("pr title should not keep the upstream ref: %s", meta.Title)
	}
}
