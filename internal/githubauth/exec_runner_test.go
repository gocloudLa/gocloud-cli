package githubauth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

// newFakeDeviceFlowServer serves the two GitHub device-flow endpoints hermetically: it returns a
// fixed device code on POST /device/code (recording the requested "scope" form value into
// requestedScope, if non-nil), and answers pendingPolls times with "authorization_pending" before
// returning accessToken on POST /token.
func newFakeDeviceFlowServer(t *testing.T, deviceCode, userCode, accessToken string, pendingPolls int, requestedScope *string) *httptest.Server {
	t.Helper()
	var polls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/device/code", func(w http.ResponseWriter, r *http.Request) {
		if requestedScope != nil {
			_ = r.ParseForm()
			*requestedScope = r.FormValue("scope")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"device_code":%q,"user_code":%q,"verification_uri":"https://github.com/login/device","expires_in":900,"interval":1}`, deviceCode, userCode)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if int(polls.Add(1)) <= pendingPolls {
			_, _ = fmt.Fprint(w, `{"error":"authorization_pending"}`)
			return
		}
		_, _ = fmt.Fprintf(w, `{"access_token":%q}`, accessToken)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// writeFakeGH creates a fake `gh` executable in dir that logs every invocation's argv (joined by
// "|", one line per invocation) to logPath, and returns fixed canned output per subcommand. This
// keeps ExecRunner tests hermetic: no real `gh` binary and no network call ever happens — we only
// ever invoke our own throwaway test fixture.
func writeFakeGH(t *testing.T, dir, logPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake gh fixture is a POSIX shell script; skip on windows")
	}

	script := `#!/bin/sh
{
  args=""
  for a in "$@"; do
    if [ -z "$args" ]; then args="$a"; else args="$args|$a"; fi
  done
  printf '%s\n' "$args" >> "` + logPath + `"
}
case "$1 $2" in
  "auth status")
    echo "  Logged in to github.com as octocat (oauth_token)" >&2
    echo "  Token scopes: 'gist', 'read:org', 'repo'" >&2
    exit 0
    ;;
  "auth token")
    echo "gho_faketoken123"
    exit 0
    ;;
  "auth login")
    cat > /dev/null
    exit 0
    ;;
  "auth setup-git")
    exit 0
    ;;
esac
if [ "$1" = "api" ]; then
  echo "gocloud-la"
  echo "another-org"
  exit 0
fi
exit 1
`
	ghPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake gh: %v", err)
	}
}

// withFakeGHOnPath prepends a directory containing a fake `gh` fixture to PATH for the duration
// of the test, restoring the original PATH via t.Cleanup.
func withFakeGHOnPath(t *testing.T) (logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "invocations.log")
	writeFakeGH(t, dir, logPath)

	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Setenv("PATH", originalPath); err != nil {
			t.Logf("warning: failed to restore PATH: %v", err)
		}
	})
	return logPath
}

func readLogLines(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read invocation log: %v", err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestExecRunner_Available(t *testing.T) {
	t.Run("gh not on PATH", func(t *testing.T) {
		originalPath := os.Getenv("PATH")
		if err := os.Setenv("PATH", t.TempDir()); err != nil {
			t.Fatalf("failed to set PATH: %v", err)
		}
		t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })

		r := &ExecRunner{}
		if r.Available() {
			t.Errorf("Available() = true, want false when gh is not on PATH")
		}
	})

	t.Run("gh on PATH", func(t *testing.T) {
		withFakeGHOnPath(t)
		r := &ExecRunner{}
		if !r.Available() {
			t.Errorf("Available() = false, want true when gh is on PATH")
		}
	})
}

// TestExecRunner_ConfigDir_SetsGHConfigDirEnv proves ConfigDir reaches every `gh` subprocess as
// GH_CONFIG_DIR — the mechanism that scopes a `gocloud sso login --provider github` session to
// the project directory (mirroring AWS_CONFIG_FILE) instead of the shared ~/.config/gh.
func TestExecRunner_ConfigDir_SetsGHConfigDirEnv(t *testing.T) {
	dir := t.TempDir()
	envLogPath := filepath.Join(dir, "env.log")
	script := `#!/bin/sh
printf '%s\n' "$GH_CONFIG_DIR" >> "` + envLogPath + `"
echo "gho_faketoken123"
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake gh: %v", err)
	}
	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })

	r := &ExecRunner{ConfigDir: filepath.Join(".", ".gh")}
	if _, err := r.Token(context.Background()); err != nil {
		t.Fatalf("Token() unexpected error: %v", err)
	}

	got, rerr := os.ReadFile(envLogPath)
	if rerr != nil {
		t.Fatalf("failed to read env log: %v", rerr)
	}
	if want := filepath.Join(".", ".gh") + "\n"; string(got) != want {
		t.Errorf("GH_CONFIG_DIR seen by gh subprocess = %q, want %q", got, want)
	}
}

// TestExecRunner_ArgvOnlyInvocation is the security-relevant test: every Runner method must
// invoke `gh` via a plain argv slice (exec.Command style), never by interpolating into a shell
// string. We prove this by inspecting the exact argv our fake gh fixture recorded: if gocloud
// were shelling out (e.g. exec.Command("sh", "-c", "gh "+strings.Join(args, " "))), the fake
// fixture would never even be reached with clean, unquoted argv — it would see "sh"/"-c" instead,
// or shell metacharacters would be free to split/inject.
func TestExecRunner_ArgvOnlyInvocation(t *testing.T) {
	logPath := withFakeGHOnPath(t)
	r := &ExecRunner{}
	ctx := context.Background()

	if _, err := r.Token(ctx); err != nil {
		t.Fatalf("Token() unexpected error: %v", err)
	}
	if _, err := r.Status(ctx); err != nil {
		t.Fatalf("Status() unexpected error: %v", err)
	}

	lines := readLogLines(t, logPath)

	assertArgvLogged := func(want string) {
		for _, l := range lines {
			if l == want {
				return
			}
		}
		t.Errorf("invocation log %v does not contain expected argv line %q", lines, want)
	}

	assertArgvLogged("auth|token")
	assertArgvLogged("auth|status")
}

// TestExecRunner_Login_DeviceFlow drives Login() end to end against a fake device-flow HTTP
// server and a fake `gh` fixture: it proves the device code is requested, the browser is opened
// immediately with no "press Enter" step, the polled token is piped into `gh auth login
// --with-token` (never `--web`, so gh's own "Authenticate Git" prompt never fires), and `gh auth
// setup-git` configures the git credential helper — all with a plain argv, never a shell string.
func TestExecRunner_Login_DeviceFlow(t *testing.T) {
	logPath := withFakeGHOnPath(t)
	var requestedScope string
	srv := newFakeDeviceFlowServer(t, "devcode-abc", "USER-CODE", "gho_devflowtoken", 1, &requestedScope)

	var openedURL string
	var out strings.Builder
	r := &ExecRunner{
		DeviceCodeURL: srv.URL + "/device/code",
		TokenURL:      srv.URL + "/token",
		OpenBrowser: func(u string) error {
			openedURL = u
			return nil
		},
		Out: &out,
	}

	if err := r.Login(context.Background()); err != nil {
		t.Fatalf("Login() unexpected error: %v", err)
	}

	if openedURL != "https://github.com/login/device" {
		t.Errorf("browser opened %q, want the verification_uri to be opened immediately (no Enter prompt)", openedURL)
	}
	if !strings.Contains(out.String(), "USER-CODE") {
		t.Errorf("Login() output = %q, want it to contain the one-time user code", out.String())
	}
	// Regression guard: `gh auth login --with-token` hard-rejects a token missing "repo" (see
	// cli/cli's internal/authflow.minimumScopes), and terraform-provider-github needs "admin:org"
	// (not just "read:org") to manage org membership, so the device flow must request all of them.
	for _, want := range []string{"repo", "read:org", "gist", "admin:org"} {
		if !strings.Contains(requestedScope, want) {
			t.Errorf("requested scope %q missing %q (gh auth login --with-token requires gh's minimum scopes)", requestedScope, want)
		}
	}

	lines := readLogLines(t, logPath)
	assertArgvLogged := func(want string) {
		for _, l := range lines {
			if l == want {
				return
			}
		}
		t.Errorf("invocation log %v does not contain expected argv line %q", lines, want)
	}
	assertArgvLogged("auth|login|--hostname|github.com|--git-protocol|https|--with-token")
	assertArgvLogged("auth|setup-git|--hostname|github.com")
}

func TestExecRunner_Login_DeviceFlow_Denied(t *testing.T) {
	withFakeGHOnPath(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/device/code", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"device_code":"dc","user_code":"UC","verification_uri":"https://github.com/login/device","expires_in":900,"interval":1}`)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"error":"access_denied"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	r := &ExecRunner{
		DeviceCodeURL: srv.URL + "/device/code",
		TokenURL:      srv.URL + "/token",
		OpenBrowser:   func(string) error { return nil },
		Out:           &strings.Builder{},
	}

	if err := r.Login(context.Background()); err == nil {
		t.Fatal("Login() error = nil, want an error when authorization is denied")
	}
}

func TestExecRunner_Token(t *testing.T) {
	withFakeGHOnPath(t)
	r := &ExecRunner{}

	got, err := r.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() unexpected error: %v", err)
	}
	if got != "gho_faketoken123" {
		t.Errorf("Token() = %q, want %q", got, "gho_faketoken123")
	}
}

func TestExecRunner_Status(t *testing.T) {
	withFakeGHOnPath(t)
	r := &ExecRunner{}

	raw, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() unexpected error: %v", err)
	}

	parsed := ParseStatus(raw)
	if !parsed.Parsed || !parsed.LoggedIn {
		t.Fatalf("ParseStatus(Status() output) = %+v, want Parsed && LoggedIn", parsed)
	}
	if parsed.Account != "octocat" {
		t.Errorf("parsed.Account = %q, want %q", parsed.Account, "octocat")
	}
	if len(parsed.Organizations) != 2 || parsed.Organizations[0] != "gocloud-la" {
		t.Errorf("parsed.Organizations = %v, want [gocloud-la another-org]", parsed.Organizations)
	}
}
