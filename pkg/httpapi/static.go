package httpapi

import "embed"

//go:embed ui/* ui/assets/*
var uiFS embed.FS
