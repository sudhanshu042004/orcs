package githubrepo

import (
	"os"
	"path/filepath"
	"testing"
)

// logTail is what the progress stream reads, so every lost or duplicated line here is a
// line the user never sees in their build log.
func newTail(t *testing.T) (*logTail, func(string)) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "1.log")
	tail := &logTail{path: path}
	t.Cleanup(tail.close)

	write := func(s string) {
		t.Helper()
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			t.Fatalf("opening log: %s", err)
		}
		defer f.Close()
		if _, err := f.WriteString(s); err != nil {
			t.Fatalf("writing log: %s", err)
		}
	}
	return tail, write
}

func wantLines(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// The worker opens the log a moment after the job is picked up, so the stream asks for
// lines before the file exists. That must not be an error.
func TestLogTailBeforeTheFileExists(t *testing.T) {
	tail, write := newTail(t)

	wantLines(t, tail.next())

	write("first\n")
	wantLines(t, tail.next(), "first")
}

func TestLogTailOnlyReturnsNewLines(t *testing.T) {
	tail, write := newTail(t)

	write("one\ntwo\n")
	wantLines(t, tail.next(), "one", "two")

	// Nothing new - the stream must not replay what it already sent
	wantLines(t, tail.next())

	write("three\n")
	wantLines(t, tail.next(), "three")
}

// A build writes a line in pieces. Emitting half of it would show the user a truncated
// line and then a second fragment that looks like its own line.
func TestLogTailHoldsBackAPartialLine(t *testing.T) {
	tail, write := newTail(t)

	write("complete\npar")
	wantLines(t, tail.next(), "complete")

	write("tial\n")
	wantLines(t, tail.next(), "partial")
}

func TestLogTailStripsCarriageReturns(t *testing.T) {
	tail, write := newTail(t)

	write("windows\r\n")
	wantLines(t, tail.next(), "windows")
}

// buildLog opens the file with O_TRUNC, so a rebuild of the same deployment restarts the
// log. A stream that was already past that point would otherwise sit at a stale offset
// and show nothing for the whole new build.
func TestLogTailRewindsWhenTheLogIsTruncated(t *testing.T) {
	tail, write := newTail(t)

	write("old build line one\nold build line two\n")
	wantLines(t, tail.next(), "old build line one", "old build line two")

	// Rebuild: the worker reopens the same path with O_TRUNC
	if err := os.Truncate(tail.path, 0); err != nil {
		t.Fatalf("truncating: %s", err)
	}
	write("new\n")

	wantLines(t, tail.next(), "new")
}

// A client that opens the stream after the build finished must get the whole log.
func TestLogTailReplaysAFinishedBuild(t *testing.T) {
	tail, write := newTail(t)

	write("one\ntwo\nthree\n")
	wantLines(t, tail.next(), "one", "two", "three")
}

// The read buffer is 32KiB, so a build that writes more than that in one go must not
// lose the overflow.
func TestLogTailReadsPastTheBufferSize(t *testing.T) {
	tail, write := newTail(t)

	const lines = 5000
	payload := ""
	want := make([]string, 0, lines)
	for i := 0; i < lines; i++ {
		line := "a line of build output that is long enough to matter"
		payload += line + "\n"
		want = append(want, line)
	}
	write(payload)

	wantLines(t, tail.next(), want...)
}
