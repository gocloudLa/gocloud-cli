package templates

import (
	_ "embed"
)

//go:embed README.md.gotmpl
var ReadmeTemplate string

//go:embed README_example.md.gotmpl
var ExampleTemplate string
