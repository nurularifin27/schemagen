package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var version = "dev"

func cliVersion() string {
	if strings.TrimSpace(version) != "" && version != "dev" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if strings.TrimSpace(info.Main.Version) != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func versionLine() string {
	return fmt.Sprintf("schemagen %s", cliVersion())
}
