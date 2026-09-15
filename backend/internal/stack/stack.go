package stack

// Stack describes a project type ORCS knows how to build.
type Stack struct {
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	Image      string   `json:"image"`
	Enabled    bool     `json:"enabled"`
	Locked     bool     `json:"locked"` // commands are fixed and cannot be overridden by the user
	InstallCmd string   `json:"install_cmd"`
	BuildCmd   string   `json:"build_cmd"`
	RunCmd     string   `json:"run_cmd"`
	OutputDirs []string `json:"output_dirs"` // searched in order for the built site
}

// supported lists every stack shown in the UI. Only React can be built today; the rest are
// advertised as coming soon so the deploy form stays honest about what it accepts.
var supported = []Stack{
	{
		Key:        "react",
		Label:      "React",
		Image:      "node:20-alpine",
		Enabled:    true,
		Locked:     true,
		InstallCmd: "npm i",
		BuildCmd:   "npm run build",
		RunCmd:     "",
		OutputDirs: []string{"dist", "build"},
	},
	{Key: "node", Label: "Node", Image: "node:20-alpine", Enabled: false},
	{Key: "go", Label: "Go", Image: "golang:1.26-alpine", Enabled: false},
	{Key: "rust", Label: "Rust", Image: "rust:1-alpine", Enabled: false},
}

// All returns every stack, enabled or not.
func All() []Stack {
	out := make([]Stack, len(supported))
	copy(out, supported)
	return out
}

// Get looks up a stack by key.
func Get(key string) (Stack, bool) {
	for _, s := range supported {
		if s.Key == key {
			return s, true
		}
	}
	return Stack{}, false
}
