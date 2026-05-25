package api

import "embed"

// all:ui embeds everything under ui/ recursively (including dotfiles
// like .gitkeep) — matches seed's canonical pattern. The previous
// two-pattern form (`ui/* ui/assets/*`) required both directories to
// have files at lint time, which broke on fresh clones.
//
//go:embed all:ui
var uiFS embed.FS
