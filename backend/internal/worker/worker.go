package worker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sudhanshu042004/orcs/internal/container"
	"github.com/sudhanshu042004/orcs/internal/queue"
	"github.com/sudhanshu042004/orcs/internal/repository"
	"github.com/sudhanshu042004/orcs/internal/s3"
	"github.com/sudhanshu042004/orcs/internal/site"
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
	fmt.Printf("[Worker] publishing builds to %s, serving them at %s\n", s3.Describe(), site.Describe())
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

// syncWriter serializes everything written to one build log. The worker writes its own
// lines while the container's output is being copied in, and both have to land in the
// order they happened.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
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

	return &syncWriter{w: file}, func() { _ = file.Close() }
}

// readyTimeout is how long a dynamic app gets to start listening before the deployment is
// called a failure.
const readyTimeout = 2 * time.Minute

// RunBuild executes one build job: spin up a container, clone and build the repository
// inside it, then publish it. A static stack has its built directory copied to object
// storage and its container thrown away; a dynamic stack keeps its container running and
// has traffic proxied to it. Nothing touches the host filesystem except the build log.
func RunBuild(job queue.Job) {
	depId := job.DeploymentId

	logFile, closeLog := buildLog(depId)
	// A dynamic deployment hands the log to a follower that outlives this function, and
	// closing it here would cut the app's own output off
	logOwned := true
	defer func() {
		if logOwned {
			closeLog()
		}
	}()

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
	installCmd, buildCmd, runCmd := job.InstallCmd, job.BuildCmd, job.RunCmd
	if target.Locked {
		installCmd, buildCmd, runCmd = target.InstallCmd, target.BuildCmd, target.RunCmd
	}
	if target.Kind == stack.KindDynamic && runCmd == "" {
		fail("A %s deployment needs a run command", target.Label)
		return
	}

	// 1. The job left the queue - the deployment is now being worked on
	if err := repository.UpdateDeploymentStatus(depId, StatusPending, ""); err != nil {
		fail("Failed to mark deployment as pending: %s", err.Error())
		return
	}
	logf("Picked up from build queue. Stack: %s (%s), %s deployment", target.Label, target.Image, target.Kind)

	// 2. Create the build container
	containerId, err = container.CreateBuildContainer(container.BuildSpec{
		DeploymentId: depId,
		Image:        target.Image,
		RepoUrl:      job.RepoUrl,
		InstallCmd:   installCmd,
		BuildCmd:     buildCmd,
		RunCmd:       runCmd,
		Port:         target.Port,
	})
	if err != nil {
		fail("Failed to create build container: %s", err.Error())
		return
	}
	if err := repository.SaveDeploymentContainer(depId, containerId); err != nil {
		fail("Failed to record build container: %s", err.Error())
		return
	}

	// 3. Run the build
	if err := container.StartContainer(containerId); err != nil {
		fail("Failed to start build container: %s", err.Error())
		return
	}

	if target.Kind == stack.KindDynamic {
		logOwned = false
		runDynamic(depId, containerId, target, logFile, logf, fail, closeLog)
		return
	}

	// A static build's container exits when the build command finishes, so its log can
	// simply be copied across until then
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

	url := site.URL(depId)
	logf("Published %d files from %s/", uploaded, published)
	logf("Deployment live at %s", url)

	_ = container.RemoveContainer(containerId)
	_ = repository.DeleteDeploymentContainer(depId)

	if err := repository.UpdateDeploymentStatus(depId, StatusDeployed, url); err != nil {
		fmt.Printf("[Worker] deployment %d: failed to record deployed url: %s\n", depId, err.Error())
	}
}

// runDynamic takes over once a dynamic stack's container has been started. The container
// installs, builds and then execs the run command, so the build and the app share one log
// and the container is expected to stay up. The deployment counts as live once the app
// accepts a connection on its port.
func runDynamic(
	depId int64,
	containerId string,
	target stack.Stack,
	logFile io.Writer,
	logf func(string, ...interface{}),
	fail func(string, ...interface{}),
	closeLog func(),
) {
	// The container never exits on its own, so its log is followed in the background and
	// goes on collecting the app's own output after the deployment is live
	stopLogs, err := container.StreamLogsTo(containerId, logFile)
	if err != nil {
		fail("Failed to follow container logs: %s", err.Error())
		closeLog()
		return
	}

	hostPort, err := awaitListening(containerId, target.Port)
	if err != nil {
		stopLogs()
		fail("%s", err.Error())
		closeLog()
		return
	}

	if err := repository.SetDeploymentHostPort(depId, hostPort); err != nil {
		stopLogs()
		fail("Failed to record the app's port: %s", err.Error())
		closeLog()
		return
	}

	// Now that the app is known to work, let docker bring it back after a restart
	if err := container.KeepAlive(containerId); err != nil {
		fmt.Printf("[Worker] deployment %d: %s\n", depId, err.Error())
	}

	url := site.URL(depId)
	logf("App is listening on port %d inside the container", target.Port)
	logf("Deployment live at %s", url)

	if err := repository.UpdateDeploymentStatus(depId, StatusDeployed, url); err != nil {
		fmt.Printf("[Worker] deployment %d: failed to record deployed url: %s\n", depId, err.Error())
	}
	// The log stays open on purpose: the app goes on writing to it while it runs
}

// readyClient is used to probe a starting app. Redirects are not followed - any answer at
// all means the app is up, and where it points is its own business.
var readyClient = &http.Client{
	Timeout: 2 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// awaitListening waits for a just started app to answer, and returns the host port it was
// published on. A container that exits first is reported as the failure it is, rather than
// being waited on until the timeout.
//
// The probe is a real request, not a TCP connect: docker publishes the host port the moment
// the container starts, so a connect succeeds long before anything inside is listening.
func awaitListening(containerId string, containerPort int) (int, error) {
	deadline := time.Now().Add(readyTimeout)
	host := site.RuntimeHost()
	hostPort := 0

	for time.Now().Before(deadline) {
		if !container.IsRunning(containerId) {
			code, err := container.GetContainerExitCode(containerId)
			if err != nil {
				return 0, fmt.Errorf("the app stopped before it started listening")
			}
			return 0, fmt.Errorf("the app exited with code %d before it started listening", code)
		}

		if hostPort == 0 {
			if port, err := container.HostPort(containerId, containerPort); err == nil {
				hostPort = port
			}
		}

		if hostPort != 0 {
			resp, err := readyClient.Get("http://" + net.JoinHostPort(host, strconv.Itoa(hostPort)) + "/")
			if err == nil {
				_ = resp.Body.Close()
				return hostPort, nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return 0, fmt.Errorf("the app did not answer on port %d within %s", containerPort, readyTimeout)
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
