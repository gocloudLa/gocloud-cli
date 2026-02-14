package utils

import (
	"testing"
)

func TestDisableColorsAndColorsEnabled(t *testing.T) {
	// DisableColors sets color.NoColor = true
	DisableColors()
	if ColorsEnabled() {
		t.Error("After DisableColors(), ColorsEnabled() should be false")
	}
	// Note: We leave colors disabled to avoid affecting other tests; color package state is process-wide.
}

func TestColorsEnabled_afterDisable(t *testing.T) {
	DisableColors()
	got := ColorsEnabled()
	if got {
		t.Error("ColorsEnabled() after DisableColors() = true, expected false")
	}
}
