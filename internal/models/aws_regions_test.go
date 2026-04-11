package models

import "testing"

func TestIsKnownAWSRegion(t *testing.T) {
	if !IsKnownAWSRegion("us-east-1") {
		t.Error("us-east-1 should be known")
	}
	if IsKnownAWSRegion("  eu-central-1  ") {
		t.Error("padded region string must not match map keys (exact key only)")
	}
	if IsKnownAWSRegion("local.metadata.aws_region") {
		t.Error("local ref should not be known region")
	}
	if IsKnownAWSRegion("us-gov-west-1") {
		t.Error("regions not in the built-in map are not known (add to AWSRegionShortCodes if needed)")
	}
}
