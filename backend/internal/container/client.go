package container

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// buildScript runs inside the build container. The repository URL and the install/build
// commands arrive as environment variables so nothing user-supplied is ever interpolated
// into the shell command line. The clone lives only inside the container.
const buildScript = `set -e
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
eval "$BUILD_CMD"`

// CreateBuildContainer creates (but does not start) the container that clones and builds a
// repository. Nothing is written to the host filesystem.
func CreateBuildContainer(depId int64, image, repoUrl, installCmd, buildCmd string) (string, error) {
	containerName := fmt.Sprintf("orcs-build-%d", depId)

	// A container from an interrupted earlier attempt would block the name
	_ = exec.Command("docker", "rm", "-f", containerName).Run()

	args := []string{
		"create",
		"--name", containerName,
		"-e", "REPO_URL=" + repoUrl,
		"-e", "INSTALL_CMD=" + installCmd,
		"-e", "BUILD_CMD=" + buildCmd,
		image, "sh", "-c", buildScript,
	}

	cmd := exec.Command("docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create docker container: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	return strings.TrimSpace(stdout.String()), nil
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
