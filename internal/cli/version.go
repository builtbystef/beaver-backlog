package cli

// This file holds the version command and the build metadata it reports. The
// metadata is injected at link time, so it is interface-owned data: it arrives
// through Env like the args and the streams, never from the core.

import (
	"fmt"

	"github.com/builtbystef/beaver-backlog/internal/output"
)

// Build is what a binary knows about how it was built. A release binary has all
// three fields injected at link time; a plain `go build` has none of them.
type Build struct {
	Version string // release version, e.g. 1.0.0
	Commit  string // commit the build came from
	Date    string // date the build was made
}

// devVersion is what a build with nothing injected calls itself: an unreleased
// binary is never mistaken for a released one.
const devVersion = "dev"

// version reports the version to show, naming an uninjected build "dev".
func (b Build) version() string {
	if b.Version == "" {
		return devVersion
	}
	return b.Version
}

// buildView is the machine shape of a build: all three fields are always
// present, empty for what an uninjected build does not know.
type buildView struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

// cmdVersion prints the build the running binary came from. It needs no store,
// so it works anywhere.
func cmdVersion(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "version")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "version takes no arguments")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	if format == output.JSON {
		if err := output.WriteJSON(env.Stdout, buildView{
			Version: env.Build.version(),
			Commit:  env.Build.Commit,
			Built:   env.Build.Date,
		}); err != nil {
			errf(env, "%v", err)
			return exitError
		}
		return exitOK
	}
	fmt.Fprintln(env.Stdout, humanVersion(env.Build))
	return exitOK
}

// humanVersion words a build as one line, mentioning only what the build knows.
func humanVersion(b Build) string {
	line := "beaver " + b.version()
	switch {
	case b.Commit != "" && b.Date != "":
		return fmt.Sprintf("%s (commit %s, built %s)", line, b.Commit, b.Date)
	case b.Commit != "":
		return fmt.Sprintf("%s (commit %s)", line, b.Commit)
	case b.Date != "":
		return fmt.Sprintf("%s (built %s)", line, b.Date)
	}
	return line
}
