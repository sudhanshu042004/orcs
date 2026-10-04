package container

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// buildScript runs inside the build container. The repository URL and the install/build
// commands arrive as environment variables so nothing user-supplied is ever interpolated
// into the shell command line. The clone lives only inside the container.
const buildScript = `set -e
# A restarted container must not clone and build a second time - the checkout is already
# there, so it goes straight back to running the app
if [ ! -f /app/.orcs-ready ]; then
  if ! command -v git >/dev/null 2>&1; then
    # Package mirrors occasionally fail to resolve, so give the install a few tries
    apk add --no-cache git >/dev/null 2>&1 ||
      (sleep 3 && apk add --no-cache git >/dev/null 2>&1) ||
      (sleep 5 && apk update >/dev/null 2>&1 && apk add --no-cache git)
  fi
  echo "> git clone $REPO_URL"
  git clone --depth 1 "$REPO_URL" /app
  cd /app
  echo "> $INSTALL_CMD"
  eval "$INSTALL_CMD"
  echo "> $BUILD_CMD"
  eval "$BUILD_CMD"
  touch /app/.orcs-ready
else
  cd /app
  echo "> restarted, reusing the existing build"
fi
if [ -n "$RUN_CMD" ]; then
  echo "> $RUN_CMD"
  # Replace the shell so the app is the container's main process and signals reach it
  exec sh -c "$RUN_CMD"
fi`

// BuildSpec is everything needed to create the container for one deployment. A static
// build leaves RunCmd empty, so the container exits once the build command finishes; a
// dynamic one sets it, and the container goes on running the app.
type BuildSpec struct {
	DeploymentId int64
	Image        string
	RepoUrl      string
	InstallCmd   string
	BuildCmd     string
	RunCmd       string
	Port         int // dynamic only: published to a random host port and passed as $PORT
}

// CreateBuildContainer creates (but does not start) the container that clones and builds a
// repository. Nothing is written to the host filesystem.
func CreateBuildContainer(spec BuildSpec) (string, error) {
	containerName := fmt.Sprintf("orcs-build-%d", spec.DeploymentId)

	// A container from an interrupted earlier attempt would block the name
	_ = exec.Command("docker", "rm", "-f", containerName).Run()

	args := []string{
		"create",
		"--name", containerName,
		"-e", "REPO_URL=" + spec.RepoUrl,
		"-e", "INSTALL_CMD=" + spec.InstallCmd,
		"-e", "BUILD_CMD=" + spec.BuildCmd,
		"-e", "RUN_CMD=" + spec.RunCmd,
	}

	// A long running app is told which port to listen on, and that port is published on
	// loopback so only this host can reach it - the site handler proxies to it
	if spec.RunCmd != "" && spec.Port > 0 {
		args = append(args,
			"-e", fmt.Sprintf("PORT=%d", spec.Port),
			"-p", fmt.Sprintf("127.0.0.1::%d", spec.Port),
		)
	}

	args = append(args, spec.Image, "sh", "-c", buildScript)

	cmd := exec.Command("docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create docker container: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// KeepAlive asks docker to bring a container back up after a daemon or host restart. It is
// applied only once an app is known to work, so a container that fails its build is never
// restarted into the same failure.
func KeepAlive(containerId string) error {
	cmd := exec.Command("docker", "update", "--restart", "unless-stopped", containerId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set restart policy: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// HostPort reports which host port docker mapped a container port to.
func HostPort(containerId string, containerPort int) (int, error) {
	cmd := exec.Command("docker", "port", containerId, fmt.Sprintf("%d/tcp", containerPort))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to read published port: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	// docker prints one "address:port" line per mapping, and may list both families
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.LastIndex(line, ":")
		if idx < 0 {
			continue
		}
		port, err := strconv.Atoi(strings.TrimSpace(line[idx+1:]))
		if err == nil && port > 0 {
			return port, nil
		}
	}
	return 0, fmt.Errorf("container %s published no host port for %d", containerId, containerPort)
}

// IsRunning reports whether a container is still up. A dynamic deployment whose container
// has exited is what tells the worker a just started app died on its own.
func IsRunning(containerId string) bool {
	cmd := exec.Command("docker", "inspect", containerId, "--format", "{{.State.Running}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(stdout.String()) == "true"
}

// StreamLogsTo follows a container's output into w and returns without waiting for it to
// finish. A dynamic container never exits on its own, so its log cannot be waited on the
// way a build can - the returned stop function ends the follow.
//
// The output is read through pipes and written a line at a time rather than handing docker
// the destination file directly: that way the container's output and the worker's own
// lines reach the log in the order they actually happened.
func StreamLogsTo(containerId string, w io.Writer) (stop func(), err error) {
	cmd := exec.Command("docker", "logs", "-f", containerId)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to follow container logs: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to follow container logs: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to follow container logs: %w", err)
	}

	var wg sync.WaitGroup
	copyLines := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			_, _ = io.WriteString(w, scanner.Text()+"\n")
		}
	}
	wg.Add(2)
	go copyLines(stdout)
	go copyLines(stderr)

	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		wg.Wait()
		_ = cmd.Wait()
	}, nil
}

// StartContainer starts a container by its ID.
func StartContainer(containerId string) error {
	cmd := exec.Command("docker", "start", containerId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// CopyTarFromContainer streams a directory out of the container as a tar archive, so build
// output can go straight to object storage without landing on the host filesystem.
// The returned wait function must be called once the reader is drained.
func CopyTarFromContainer(containerId string, srcPath string) (io.ReadCloser, func() error, error) {
	cmd := exec.Command("docker", "cp", fmt.Sprintf("%s:%s", containerId, srcPath), "-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open docker cp stream: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start docker cp: %w", err)
	}

	wait := func() error {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("docker cp failed: %s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return nil
	}

	return stdout, wait, nil
}

// RemoveContainer removes a container by its ID.
func RemoveContainer(containerId string) error {
	cmd := exec.Command("docker", "rm", "-f", containerId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// GetContainerExitCode gets the status code of a container.
func GetContainerExitCode(containerId string) (int, error) {
	cmd := exec.Command("docker", "inspect", containerId, "--format", "{{.State.ExitCode}}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return -1, fmt.Errorf("failed to inspect container state: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	exitCodeStr := strings.TrimSpace(stdout.String())
	var exitCode int
	if _, err := fmt.Sscanf(exitCodeStr, "%d", &exitCode); err != nil {
		return -1, fmt.Errorf("failed to parse exit code %q: %w", exitCodeStr, err)
	}

	return exitCode, nil
}
