package web

import "embed"

// Dist holds the built frontend; "all:" keeps underscore/dot-prefixed assets.
//
//go:embed all:dist
var Dist embed.FS
