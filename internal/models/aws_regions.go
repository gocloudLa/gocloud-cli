package models

// AWSRegionShortCodes maps canonical AWS region names to short codes used in metadata.tf (same set as generator mapRegionToShortCode).
var AWSRegionShortCodes = map[string]string{
	// US Regions
	"us-east-1": "use1",
	"us-east-2": "use2",
	"us-west-1": "usw1",
	"us-west-2": "usw2",

	// Europe Regions
	"eu-west-1":    "euw1",
	"eu-west-2":    "euw2",
	"eu-west-3":    "euw3",
	"eu-central-1": "euc1",
	"eu-central-2": "euc2",
	"eu-north-1":   "eun1",
	"eu-south-1":   "eus1",
	"eu-south-2":   "eus2",

	// Asia Pacific Regions
	"ap-southeast-1": "apse1",
	"ap-southeast-2": "apse2",
	"ap-southeast-3": "apse3",
	"ap-southeast-4": "apse4",
	"ap-northeast-1": "apne1",
	"ap-northeast-2": "apne2",
	"ap-northeast-3": "apne3",
	"ap-south-1":     "aps1",
	"ap-south-2":     "aps2",
	"ap-east-1":      "ape1",

	// Canada Regions
	"ca-central-1": "cac1",
	"ca-west-1":    "caw1",

	// South America Regions
	"sa-east-1": "sae1",

	// Africa Regions
	"af-south-1": "afs1",

	// Middle East Regions
	"me-south-1":   "mes1",
	"me-central-1": "mec1",

	// Israel Regions
	"il-central-1": "ilc1",
}

// IsKnownAWSRegion reports whether region is an exact key in AWSRegionShortCodes.
func IsKnownAWSRegion(region string) bool {
	_, ok := AWSRegionShortCodes[region]
	return ok
}
