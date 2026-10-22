package assets

import "embed"

// FS is the pinned offline browser runtime and its license notices.
//
//go:embed browser/**
var FS embed.FS
