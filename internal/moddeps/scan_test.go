package moddeps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSemver(t *testing.T) {
	a := ParseSemver("1.2.3-alpha")
	b := ParseSemver("1.2.3")
	if a != b {
		t.Fatalf("pre-release strip: got %+v want %+v", a, b)
	}
}

func TestCmpVersion(t *testing.T) {
	if CmpVersion("1.0.0", "2.0.0") >= 0 {
		t.Fatal("expect current < latest")
	}
	if CmpVersion("2.0.0", "2.0.0") != 0 {
		t.Fatal("equal")
	}
}

func TestLatestVersion(t *testing.T) {
	got := LatestVersion([]string{"1.0.0", "1.2.0", "1.1.0"})
	if got != "1.2.0" {
		t.Fatalf("got %q", got)
	}
}

func TestSemverBumpLevel(t *testing.T) {
	if SemverBumpLevel("1.0.0", "2.0.0") != "major" {
		t.Fatal()
	}
	if SemverBumpLevel("1.0.0", "1.1.0") != "minor" {
		t.Fatal()
	}
	if SemverBumpLevel("1.0.0", "1.0.1") != "patch" {
		t.Fatal()
	}
}

func TestCollectTFDeps(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "main.tf")
	content := `
module "x" {
  source  = "hashicorp/consul/aws"
  version = "0.11.0"
}
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}
`
	if err := os.WriteFile(tf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, provs, err := CollectTFDeps(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Source != "hashicorp/consul/aws" {
		t.Fatalf("modules: %+v", mods)
	}
	if len(provs) != 1 || provs[0].Source != "hashicorp/aws" {
		t.Fatalf("providers: %+v", provs)
	}
}

func TestApplyModuleVersionBump(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.tf")
	content := `module "x" {
  source  = "hashicorp/consul/aws"
  version = "0.11.0"
}
`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ApplyModuleVersionBump(dir, "hashicorp/consul/aws", "0.11.0", "0.12.0", "m.tf")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	out, _ := os.ReadFile(p)
	if string(out) == content {
		t.Fatal("expected rewrite")
	}
}
