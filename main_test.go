package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionFromBuildInfo(t *testing.T) {
	for _, tc := range []struct {
		name string
		info debug.BuildInfo
		want [3]string
	}{
		{
			name: "go install of a tag",
			info: debug.BuildInfo{Main: debug.Module{Version: "v1.2.0"}},
			want: [3]string{"v1.2.0", "none", "unknown"},
		},
		{
			name: "local build with VCS stamp",
			info: debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "aa9b772f00c0ffee"},
					{Key: "vcs.time", Value: "2026-11-19T08:34:00Z"},
				},
			},
			want: [3]string{"dev", "aa9b772", "2026-11-19T08:34:00Z"},
		},
		{
			name: "nothing known",
			want: [3]string{"dev", "none", "unknown"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, c, d := versionFromBuildInfo(&tc.info, "dev", "none", "unknown")
			if got := [3]string{v, c, d}; got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
