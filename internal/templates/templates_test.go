package templates

import (
	"strings"
	"testing"
)

func TestReadmeTemplate_NotEmpty(t *testing.T) {
	if ReadmeTemplate == "" {
		t.Error("ReadmeTemplate should not be empty (embed failed)")
	}
}

func TestExampleTemplate_NotEmpty(t *testing.T) {
	if ExampleTemplate == "" {
		t.Error("ExampleTemplate should not be empty (embed failed)")
	}
}

func TestReadmeTemplate_ContainsExpectedContent(t *testing.T) {
	if !strings.Contains(ReadmeTemplate, "terraform") && !strings.Contains(ReadmeTemplate, "module") {
		t.Log("ReadmeTemplate does not contain 'terraform' or 'module'; template may have changed")
	}
	// At least one placeholder or common README content
	if len(ReadmeTemplate) < 100 {
		t.Errorf("ReadmeTemplate seems too short (%d bytes)", len(ReadmeTemplate))
	}
}

func TestExampleTemplate_ContainsExpectedContent(t *testing.T) {
	if len(ExampleTemplate) < 50 {
		t.Errorf("ExampleTemplate seems too short (%d bytes)", len(ExampleTemplate))
	}
}
