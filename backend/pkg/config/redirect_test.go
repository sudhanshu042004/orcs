package config

import "testing"

func TestSafeRedirectPathRejectsOffsiteTargets(t *testing.T) {
	// Anything that could send a logged-in user somewhere other than the frontend
	offsite := []string{
		"", "dashboard", "http://evil.com", "https://evil.com/x",
		"//evil.com", "//evil.com/path", "/\\evil.com", "javascript:alert(1)",
	}
	for _, in := range offsite {
		if got := SafeRedirectPath(in); got != defaultRedirectPath {
			t.Errorf("SafeRedirectPath(%q) = %q, want the default %q", in, got, defaultRedirectPath)
		}
	}

	onsite := []string{"/", "/dashboard", "/project/12", "/dashboard/deploy?name=x"}
	for _, in := range onsite {
		if got := SafeRedirectPath(in); got != in {
			t.Errorf("SafeRedirectPath(%q) = %q, want it kept as-is", in, got)
		}
	}
}

func TestStateRoundTrip(t *testing.T) {
	for _, path := range []string{"/", "/dashboard", "/project/12?tab=logs"} {
		got, ok := DecodeState(EncodeState(path))
		if !ok || got != path {
			t.Errorf("round trip of %q gave (%q, %v)", path, got, ok)
		}
	}
}

func TestDecodeStateRejectsBadState(t *testing.T) {
	bad := []string{
		"",                      // github sent nothing back
		"randomstate",           // no payload
		"wrongnonce:L2Rhc2g",    // nonce does not match - not a state we issued
		"randomstate:!!!notb64", // payload is not base64
	}
	for _, state := range bad {
		if path, ok := DecodeState(state); ok {
			t.Errorf("DecodeState(%q) accepted it and returned %q", state, path)
		}
	}
}

func TestDecodeStateSanitizesPayload(t *testing.T) {
	// A state we signed is still not trusted to carry a safe path
	state := EncodeState("/dashboard")
	hostile := EncodeState("//evil.com")
	if state == hostile {
		t.Fatal("test setup is wrong")
	}
	got, ok := DecodeState(hostile)
	if !ok {
		t.Fatal("a well-formed state should decode")
	}
	if got != defaultRedirectPath {
		t.Errorf("DecodeState carried an offsite path through: %q", got)
	}
}
