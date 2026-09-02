package cmd

import (
	"reflect"
	"testing"
)

func TestParseProviderFlag(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		want        []string
		expectError bool
	}{
		{name: "empty means unrestricted", value: "", want: nil},
		{name: "aws only", value: "aws", want: []string{"aws"}},
		{name: "github only", value: "github", want: []string{"github"}},
		{name: "aws,github combined", value: "aws,github", want: []string{"aws", "github"}},
		{name: "github,aws order independent", value: "github,aws", want: []string{"aws", "github"}},
		{name: "all is an alias for aws+github", value: "all", want: []string{"aws", "github"}},
		{name: "unknown provider name is rejected", value: "bitbucket", expectError: true},
		{name: "duplicate provider name is rejected", value: "aws,aws", expectError: true},
		{name: "trailing comma is rejected", value: "aws,", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProviderFlag(tt.value)
			if tt.expectError {
				if err == nil {
					t.Fatalf("parseProviderFlag(%q) expected error, got nil (result: %v)", tt.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseProviderFlag(%q) unexpected error: %v", tt.value, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseProviderFlag(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolveSSOProviders(t *testing.T) {
	tests := []struct {
		name      string
		declared  []string
		flagValue string
		want      []string
	}{
		{
			name:      "empty flag defers entirely to declared providers (aws only)",
			declared:  []string{"aws"},
			flagValue: "",
			want:      []string{"aws"},
		},
		{
			name:      "empty flag defers entirely to declared providers (aws+github)",
			declared:  []string{"aws", "github"},
			flagValue: "",
			want:      []string{"aws", "github"},
		},
		{
			name:      "flag narrows declared set to aws only",
			declared:  []string{"aws", "github"},
			flagValue: "aws",
			want:      []string{"aws"},
		},
		{
			name:      "flag requests github but it is not declared in config: resolves to empty",
			declared:  []string{"aws"},
			flagValue: "github",
			want:      nil,
		},
		{
			name:      "all resolves to the intersection with declared",
			declared:  []string{"aws"},
			flagValue: "all",
			want:      []string{"aws"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveSSOProviders(tt.declared, tt.flagValue)
			if err != nil {
				t.Fatalf("resolveSSOProviders() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveSSOProviders(%v, %q) = %v, want %v", tt.declared, tt.flagValue, got, tt.want)
			}
		})
	}
}
