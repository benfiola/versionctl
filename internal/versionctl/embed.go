package versionctl

import _ "embed"

//go:embed default-config.json
var DefaultConfig []byte

//go:embed version.txt
var VersionctlVersion string
