package version

import (
    "fmt"
    "runtime"
)

type Info struct {
/*    GitTag       string `json:"gitTag"`
    GitBranch    string `json:"gitBranch"`
    GitCommit    string `json:"gitCommit"`
    GitTreeState string `json:"gitTreeState"`
    BuildDate    string `json:"buildDate"` */
    minibpVersion    string "0.001"
    GoVersion    string `json:"goVersion"`
    Compiler     string `json:"compiler"`
    Platform     string `json:"platform"`
}

/*
func (info Info) String() string {
    return info.GitTag
} */

func minibpVersion() Info {
    return Info{
/*        GitTag:       gitTag,
        GitBranch:    gitBranch,
        GitCommit:    gitCommit,
        GitTreeState: gitTreeState,
        BuildDate:    buildDate, */
	minibpVersion:    "0.001",
        GoVersion:    runtime.Version(),
        Compiler:     runtime.Compiler,
        Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
    }
}
