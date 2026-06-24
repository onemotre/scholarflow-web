package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticEmbed embed.FS

// StaticFS is the embedded static asset tree rooted at "static".
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticEmbed, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
