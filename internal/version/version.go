package version

import (
	"runtime/debug"
	"strings"
)

// Version is populated at build time using -ldflags.
var Version = ""

type Info struct {
	Version  string
	Commit   string
	Modified bool
}

func Current() Info {
	v := Info{Version: Version}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				v.Commit = setting.Value
			case "vcs.modified":
				v.Modified = setting.Value == "true"
			}
		}
	}

	return v
}

func (v Info) String() string {
	parts := []string{}

	if v.Version != "" {
		parts = append(parts, v.Version)
	}

	if v.Commit != "" {
		commit := v.Commit
		if len(commit) > 12 {
			commit = commit[:12]
		}
		parts = append(parts, "commit "+commit)
	}

	if len(parts) == 0 {
		return "unknown"
	}

	result := strings.Join(parts, " (")
	if len(parts) > 1 {
		result += ")"
	}

	if v.Modified {
		result += " (modified)"
	}

	return result
}
