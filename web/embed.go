package web

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/*
var DistFS embed.FS

func SubDist() (fs.FS, error) {
	return fs.Sub(DistFS, "dist")
}
