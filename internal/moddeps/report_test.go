package moddeps

import "testing"

func TestBumpBranchStablePerPath(t *testing.T) {
	a := bumpBranch("gocloudLa/x/aws", "1.2.0", "modules/base/main.tf")
	b := bumpBranch("gocloudLa/x/aws", "1.2.0", "modules/base/main.tf")
	if a != b {
		t.Fatalf("same inputs: got %q vs %q", a, b)
	}
	c := bumpBranch("gocloudLa/x/aws", "1.2.0", "modules/other/main.tf")
	if a == c {
		t.Fatal("different path must produce different branch")
	}
}
