package stack

import "testing"

// The stack table is the single source of truth for every build, so a malformed entry is
// a deploy that fails halfway through rather than a request that is rejected. These are
// the invariants the worker and the site handler rely on.
func TestSupportedStacksAreWellFormed(t *testing.T) {
	seen := map[string]bool{}

	for _, s := range supported {
		t.Run(s.Key, func(t *testing.T) {
			if s.Key == "" || s.Label == "" {
				t.Fatal("key and label are required")
			}
			if seen[s.Key] {
				t.Fatalf("duplicate key %q", s.Key)
			}
			seen[s.Key] = true

			if !s.Enabled {
				// A coming-soon entry only has to be listable
				return
			}

			if s.Image == "" {
				t.Error("an enabled stack needs an image")
			}
			if s.Kind != KindStatic && s.Kind != KindDynamic {
				t.Errorf("kind %q is neither static nor dynamic", s.Kind)
			}
			if s.InstallCmd == "" || s.BuildCmd == "" {
				t.Error("install and build commands are required, they pre-fill the deploy form")
			}

			switch s.Kind {
			case KindStatic:
				if len(s.OutputDirs) == 0 {
					t.Error("a static stack must say where its build output lands")
				}
				if s.RunCmd != "" {
					t.Error("a static stack's container exits after the build; RunCmd is never used")
				}
			case KindDynamic:
				if s.RunCmd == "" {
					t.Error("a dynamic stack must know how to start the app")
				}
				if s.Port <= 0 {
					t.Error("a dynamic stack must declare the port it is proxied to")
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	if _, ok := Get("react"); !ok {
		t.Error("react should be a known stack")
	}
	if _, ok := Get("nope"); ok {
		t.Error("an unknown key should not resolve")
	}
	if _, ok := Get("React"); ok {
		t.Error("lookup should be exact - a near miss must be rejected, not guessed at")
	}
}

// All hands the table to the deploy form, so a caller must not be able to edit the
// process-wide config through it.
func TestAllReturnsACopy(t *testing.T) {
	first := All()
	if len(first) != len(supported) {
		t.Fatalf("All returned %d stacks, want %d", len(first), len(supported))
	}

	first[0].BuildCmd = "rm -rf /"
	if All()[0].BuildCmd == "rm -rf /" {
		t.Error("mutating the result of All changed the shared stack table")
	}
}
