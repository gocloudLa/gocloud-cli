package githubauth

import (
	"context"
	"errors"
	"testing"
)

// TestFakeRunner_ImplementsRunner is a compile-time-ish check that FakeRunner satisfies Runner,
// exercised via a var assignment plus behavioral assertions on each configurable method.
func TestFakeRunner_ImplementsRunner(t *testing.T) {
	var _ Runner = &FakeRunner{}
}

func TestFakeRunner_Available(t *testing.T) {
	f := &FakeRunner{AvailableResult: true}
	if !f.Available() {
		t.Errorf("Available() = false, want true")
	}

	f2 := &FakeRunner{AvailableResult: false}
	if f2.Available() {
		t.Errorf("Available() = true, want false")
	}
}

func TestFakeRunner_Login(t *testing.T) {
	wantErr := errors.New("boom")
	f := &FakeRunner{LoginErr: wantErr}
	if err := f.Login(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("Login() error = %v, want %v", err, wantErr)
	}
	if !f.LoginCalled {
		t.Errorf("Login() did not record LoginCalled")
	}
}

func TestFakeRunner_Token(t *testing.T) {
	f := &FakeRunner{TokenResult: "gho_abc123"}
	got, err := f.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() unexpected error: %v", err)
	}
	if got != "gho_abc123" {
		t.Errorf("Token() = %q, want %q", got, "gho_abc123")
	}
}

func TestFakeRunner_Status(t *testing.T) {
	f := &FakeRunner{StatusResult: "raw status text"}
	got, err := f.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() unexpected error: %v", err)
	}
	if got != "raw status text" {
		t.Errorf("Status() = %q, want %q", got, "raw status text")
	}
}
