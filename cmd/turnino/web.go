package main

import "embed"

// webFiles contains the complete browser UI so releases remain single binaries.
//
//go:embed web/templates/*.html web/static/*
var webFiles embed.FS
