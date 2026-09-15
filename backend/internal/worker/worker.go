package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sudhanshu042004/orcs/internal/container"
	"github.com/sudhanshu042004/orcs/internal/queue"
	"github.com/sudhanshu042004/orcs/internal/repository"
	"github.com/sudhanshu042004/orcs/internal/s3"
	"github.com/sudhanshu042004/orcs/internal/stack"
)

const (
	StatusQueued   = "queued"
	StatusPending  = "pending"
	StatusDeployed = "deployed"
	StatusFailed   = "failed"
)

// Start wires the build queue up to the worker pool and puts any job that was still
// queued when the process last stopped back on the queue.
func Start(workers int) {
	fmt.Printf("[Worker] publishing builds to %s\n", s3.Describe())
	queue.Default().Start(workers, RunBuild)

	queued, err := repository.GetQueuedDeployments()
	if err != nil {
		fmt.Printf("[Worker] failed to recover queued deployments: %s\n", err.Error())
		return
	}
	for _, dep := range queued {
		job := queue.Job{
			DeploymentId: dep.Id,
			Stack:        dep.Stack,
			RepoUrl:      dep.RepoUrl,
			InstallCmd:   dep.InstallCmd,
			BuildCmd:     dep.BuildCmd,
			RunCmd:       dep.RunCmd,
		}
		if err := queue.Default().Enqueue(job); err != nil {
			fmt.Printf("[Worker] could not re-queue deployment %d: %s\n", dep.Id, err.Error())
			_ = repository.UpdateDeploymentStatus(dep.Id, StatusFailed, "")
			continue
		}
		fmt.Printf("[Worker] recovered queued deployment %d\n", dep.Id)
	}
}

// buildLog opens the log the progress stream replays for a deployment. A log that cannot be
// written is never fatal - the build still runs, it just has no history to replay.
func buildLog(depId int64) (io.Writer, func()) {
	logDir := filepath.Join("uploads", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("[Worker] deployment %d: build log unavailable: %s\n", depId, err.Error())
		return io.Discard, func() {}
	}

	file, err := os.OpenFile(filepath.Join(logDir, fmt.Sprintf("%d.log", depId)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("[Worker] deployment %d: build log unavailable: %s\n", depId, err.Error())
		return io.Discard, func() {}
	}

	return file, func() { _ = file.Close() }
}

// RunBuild executes one build job: spin up a container, clone and build the repository
// inside it, publish the built site to object storage. Nothing touches the host filesystem
// except the build log.
func RunBuild(job queue.Job) {
	depId := job.DeploymentId

	logFile, closeLog := buildLog(depId)
	defer closeLog()

	logf := func(format string, args ...interface{}) {
		line := fmt.Sprintf(format, args...)
		fmt.Printf("[Worker] deployment %d: %s\n", depId, line)
		_, _ = io.WriteString(logFile, line+"\n")
	}

	var containerId string
	fail := func(format string, args ...interface{}) {
		logf(format, args...)
		if containerId != "" {
			_ = container.RemoveContainer(containerId)
			_ = repository.DeleteDeploymentContainer(depId)
		}
		_ = repository.UpdateDeploymentStatus(depId, StatusFailed, "")
	}

	var err error
	target, ok := stack.Get(job.Stack)
	if !ok || !target.Enabled {
		fail("Unsupported stack %q", job.Stack)
		return
	}

	// Locked stacks always build with their own commands, whatever was stored on the row
	installCmd, buildCmd := job.InstallCmd, job.BuildCmd
	if target.Locked {
		installCmd, buildCmd = target.InstallCmd, target.BuildCmd
	}

	// 1. The job left the queue - the deployment is now being worked on
	if err := repository.UpdateDeploymentStatus(depId, StatusPending, ""); err != nil {
		fail("Failed to mark deployment as pending: %s", err.Error())
		return
	}
	logf("Picked up from build queue. Stack: %s (%s)", target.Label, target.Image)

	// 2. Create the build container
	containerId, err = container.CreateBuildContainer(depId, target.Image, job.RepoUrl, installCmd, buildCmd)
	if err != nil {
		fail("Failed to create build container: %s", err.Error())
		return
	}
	if err := repository.SaveDeploymentContainer(depId, containerId); err != nil {
		fail("Failed to record build container: %s", err.Error())
		return
	}

	// 3. Run the build and copy its output into the log as it happens
	if err := container.StartContainer(containerId); err != nil {
		fail("Failed to start build container: %s", err.Error())
		return
	}

	logCmd := exec.Command("docker", "logs", "-f", containerId)
	logCmd.Stdout = logFile
	logCmd.Stderr = logFile
	if err := logCmd.Run(); err != nil {
		fmt.Printf("[Worker] deployment %d: log streaming ended with %s\n", depId, err.Error())
	}

	exitCode, err := container.GetContainerExitCode(containerId)
	if err != nil {
		fail("Could not read build exit code: %s", err.Error())
		return
	}
	if exitCode != 0 {
		fail("Build failed with exit code %d", exitCode)
		return
	}

	// 4. Stream the built site straight from the container into object storage
	s3Prefix := fmt.Sprintf("deployments/%d", depId)
	logf("Build succeeded. Publishing site...")

	uploaded := 0
	published := ""
	for _, dir := range target.OutputDirs {
		count, err := publishDir(containerId, "/app/"+dir, s3Prefix)
		if err != nil {
			fail("Failed to publish %s/: %s", dir, err.Error())
			return
		}
		if count > 0 {
			uploaded, published = count, dir
			break
		}
		logf("No %s/ directory in the build output, trying next...", dir)
	}

	if uploaded == 0 {
		fail("Build produced no output in %v", target.OutputDirs)
		return
	}

	url := s3.PublicURL(s3Prefix + "/index.html")
	logf("Published %d files from %s/", uploaded, published)
	logf("Deployment live at %s", url)

	_ = container.RemoveContainer(containerId)
	_ = repository.DeleteDeploymentContainer(depId)

	if err := repository.UpdateDeploymentStatus(depId, StatusDeployed, url); err != nil {
		fmt.Printf("[Worker] deployment %d: failed to record deployed url: %s\n", depId, err.Error())
	}
}

// publishDir copies one directory out of the container as a tar stream and uploads its
// contents. A directory the build never produced yields zero files rather than an error;
// anything that actually went wrong (storage unreachable, a failed upload) is reported.
func publishDir(containerId string, srcPath string, s3Prefix string) (int, error) {
	stream, wait, err := container.CopyTarFromContainer(containerId, srcPath)
	if err != nil {
		return 0, err
	}

	// The archive is rooted at the directory itself, so drop that first segment
	uploaded, uploadErr := s3.UploadTarStream(context.Background(), stream, s3Prefix, 1)
	_ = stream.Close()
	waitErr := wait()

	if uploadErr != nil {
		return uploaded, uploadErr
	}
	if waitErr != nil {
		if uploaded == 0 {
			// docker cp reports a missing path here - let the caller try the next candidate
			return 0, nil
		}
		return uploaded, waitErr
	}
	return uploaded, nil
}
