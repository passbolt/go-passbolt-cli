package main

import (
	"runtime/debug"
	"time"
)

var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
	dirty   = false
)

func init() {
	// if not set by goreleaser, use buildinfo instead
	if version == "unknown" {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		if info.Main.Version != "" {
			version = info.Main.Version
		}

		for _, kv := range info.Settings {
			if kv.Value == "" {
				continue
			}
			switch kv.Key {
			case "vcs.revision":
				commit = kv.Value
			case "vcs.time":
				d, _ := time.Parse(time.RFC3339, kv.Value)
				date = d.String()
			case "vcs.modified":
				dirty = kv.Value == "true"
			}
		}
	}
}
