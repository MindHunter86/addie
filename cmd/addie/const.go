package main

import "github.com/MindHunter86/addie/internal/utils"

var (
	buildtime = "never"

	name    = "addie"
	version = utils.DevelVersionIdent // -ldflags="-X main.version=X.X.X"
	usage   = "AniLibria media delivery manager (ADDIE)"

	copyright = "(c) 2022-2026 mindhunter86\nwith love for AniLibria (AniLiberty) project"
)
