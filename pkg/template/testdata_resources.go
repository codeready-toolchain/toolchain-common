package template

import "embed"

//go:embed testdata/*
var EFS embed.FS

//go:embed testdata/host/*
var HostFS embed.FS

//go:embed testdata/member/*
var MemberFS embed.FS
