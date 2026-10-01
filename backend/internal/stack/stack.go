package stack

// Kind decides what happens once a build succeeds.
type Kind string

const (
	// KindStatic builds to a directory of files, which is published to object storage
	// and served from there. Nothing keeps running afterwards.
	KindStatic Kind = "static"
	// KindDynamic builds an application that then has to stay up. Its container keeps
	// running and requests for the site are proxied to it.
	KindDynamic Kind = "dynamic"
)

// Stack describes a project type ORCS knows how to build.
type Stack struct {
	Key        string   `json:"key"`
	Kind       Kind     `json:"kind"`
	Label      string   `json:"label"`
	Image      string   `json:"image"`
	Enabled    bool     `json:"enabled"`
	Locked     bool     `json:"locked"` // commands are fixed and cannot be overridden by the user
	InstallCmd string   `json:"install_cmd"`
	BuildCmd   string   `json:"build_cmd"`
	RunCmd     string   `json:"run_cmd"`
	OutputDirs []string `json:"output_dirs"` // static only: searched in order for the built site
	Port       int      `json:"port"`        // dynamic only: the port the app is told to listen on
}

// defaultPort is what every dynamic stack is told to listen on, through $PORT.
const defaultPort = 3000

// supported lists every stack shown in the UI. Each one carries the commands it is built
// with, so picking a project type on the deploy form is enough to configure a build; a
// stack that is not Locked offers those commands as a starting point the user can edit.
var supported = []Stack{
	{
		Key:        "react",
		Kind:       KindStatic,
		Label:      "React",
		Image:      "node:20-alpine",
		Enabled:    true,
		Locked:     true,
		InstallCmd: "npm i",
		BuildCmd:   "npm run build",
		RunCmd:     "",
		OutputDirs: []string{"dist", "build"},
	},
	{
		Key:        "node",
		Kind:       KindDynamic,
		Label:      "Node",
		Image:      "node:20-alpine",
		Enabled:    true,
		Locked:     false,
		InstallCmd: "npm i",
		// Plenty of node services have no build step at all
		BuildCmd: "npm run build --if-present",
		RunCmd:   "npm start",
		Port:     defaultPort,
	},
	{
		Key:        "go",
		Kind:       KindDynamic,
		Label:      "Go",
		Image:      "golang:1.26",
		Enabled:    true,
		Locked:     false,
		InstallCmd: "go mod download",
		BuildCmd:   "go build -o /tmp/server .",
		RunCmd:     "/tmp/server",
		Port:       defaultPort,
	},
	{
		Key:        "rust",
		Kind:       KindDynamic,
		Label:      "Rust",
		Image:      "rust:latest",
		Enabled:    true,
		Locked:     false,
		InstallCmd: "cargo fetch",
		BuildCmd:   "cargo build --release",
		// The release binary is already built, so this only starts it
		RunCmd: "cargo run --release",
		Port:   defaultPort,
	},
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
