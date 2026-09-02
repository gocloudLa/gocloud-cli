package githubauth

import "context"

// FakeRunner is a hermetic test double for Runner: no process is ever spawned, no network call
// is ever made. Tests configure the *Result/*Err fields and assert on the *Called flags.
type FakeRunner struct {
	AvailableResult bool

	LoginErr    error
	LoginCalled bool

	TokenResult string
	TokenErr    error
	TokenCalled bool

	StatusResult string
	StatusErr    error
	StatusCalled bool
}

func (f *FakeRunner) Available() bool {
	return f.AvailableResult
}

func (f *FakeRunner) Login(ctx context.Context) error {
	f.LoginCalled = true
	return f.LoginErr
}

func (f *FakeRunner) Token(ctx context.Context) (string, error) {
	f.TokenCalled = true
	return f.TokenResult, f.TokenErr
}

func (f *FakeRunner) Status(ctx context.Context) (string, error) {
	f.StatusCalled = true
	return f.StatusResult, f.StatusErr
}
