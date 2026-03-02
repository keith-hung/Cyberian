package cmd

// Set via ldflags at build time.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type versionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// RunVersion prints version information.
func RunVersion(gf *GlobalFlags) {
	OutputJSON(versionOutput{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}, gf.Pretty)
}
