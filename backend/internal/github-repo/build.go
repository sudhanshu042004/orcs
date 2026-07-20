package githubrepo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sudhanshu042004/orcs/internal/repository"
)

// runIsolatedCommand runs the specified command inside a secure sandbox jail.
// It creates a private mount and PID namespace, bind-mounts the targetDir as read-write,
// bind-mounts core host system files (/usr, /bin, /lib, /etc, /home) as read-only,
// mounts the /proc filesystem, and then chroots to run the command inside the jail.
func runIsolatedCommand(depId int64, targetDir string, command string) error {
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	sandboxRoot := fmt.Sprintf("/tmp/orcs-sandbox-%d", depId)

	// Determine available host directories to mount read-only
	var mounts []string
	mounts = append(mounts, "usr", "bin", "lib", "etc")
	if _, err := os.Stat("/lib64"); err == nil {
		mounts = append(mounts, "lib64")
	}
	if _, err := os.Stat("/home"); err == nil {
		mounts = append(mounts, "home")
	}

	var scriptParts []string
	scriptParts = append(scriptParts, "set -e")

	// 1. Create target sandbox mount points
	mkdirCmd := "mkdir -p " + filepath.Join(sandboxRoot, "app") + " " + filepath.Join(sandboxRoot, "proc")
	for _, m := range mounts {
		mkdirCmd += " " + filepath.Join(sandboxRoot, m)
	}
	scriptParts = append(scriptParts, mkdirCmd)

	// 2. Bind-mount project directory (Read-Write)
	scriptParts = append(scriptParts, fmt.Sprintf("mount --bind %s %s", absTargetDir, filepath.Join(sandboxRoot, "app")))

	// 3. Bind-mount system directories (Read-Only)
	for _, m := range mounts {
		targetMount := filepath.Join(sandboxRoot, m)
		scriptParts = append(scriptParts, fmt.Sprintf("mount --bind /%s %s", m, targetMount))
		scriptParts = append(scriptParts, fmt.Sprintf("mount -o remount,ro,bind %s", targetMount))
	}

	// 4. Mount private proc filesystem
	scriptParts = append(scriptParts, fmt.Sprintf("mount -t proc proc %s", filepath.Join(sandboxRoot, "proc")))

	// 5. Chroot and execute command inside chroot jail
	scriptParts = append(scriptParts, fmt.Sprintf("chroot %s sh -c \"cd /app && %s\"", sandboxRoot, command))

	script := strings.Join(scriptParts, "\n")

	// Execute mount and PID namespaces
	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	runErr := cmd.Run()

	// Clean up temporary sandbox directory on the host (mounts are auto-unmounted by kernel)
	if err := os.RemoveAll(sandboxRoot); err != nil {
		fmt.Printf("[Sandbox] Warning: failed to clean up sandbox root %s: %s\n", sandboxRoot, err.Error())
	}

	return runErr
}

// BuildProjectAsync starts a background goroutine to build the project.
// It updates the database status to 'building', performs 'npm install',
// then 'npm run build', and updates the status to 'success' or 'failed'.
func BuildProjectAsync(depId int64, targetDir string) {
	go func() {
		fmt.Printf("[Build] Starting background build for deployment %d in directory %s\n", depId, targetDir)

		// 1. Update database status to 'building'
		if err := repository.UpdateDeploymentStatus(depId, "building", targetDir); err != nil {
			fmt.Printf("[Build] Error updating status to building for deployment %d: %s\n", depId, err.Error())
			return
		}

		// Check if package.json exists in targetDir
		packageJsonPath := filepath.Join(targetDir, "package.json")
		if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
			fmt.Printf("[Build] Error for deployment %d: package.json not found in %s\n", depId, targetDir)
			_ = repository.UpdateDeploymentStatus(depId, "failed", targetDir)
			return
		}

		// 2. Run npm install inside isolated chroot sandbox jail
		fmt.Printf("[Build] Running npm install in %s...\n", targetDir)
		if err := runIsolatedCommand(depId, targetDir, "npm install"); err != nil {
			fmt.Printf("[Build] npm install failed for deployment %d: %s\n", depId, err.Error())
			_ = repository.UpdateDeploymentStatus(depId, "failed", targetDir)
			return
		}

		// 3. Run npm run build inside isolated chroot sandbox jail
		fmt.Printf("[Build] Running npm run build in %s...\n", targetDir)
		if err := runIsolatedCommand(depId, targetDir, "npm run build"); err != nil {
			fmt.Printf("[Build] npm run build failed for deployment %d: %s\n", depId, err.Error())
			_ = repository.UpdateDeploymentStatus(depId, "failed", targetDir)
			return
		}

		// 4. Update status to 'success'
		fmt.Printf("[Build] Build succeeded for deployment %d!\n", depId)
		if err := repository.UpdateDeploymentStatus(depId, "success", targetDir); err != nil {
			fmt.Printf("[Build] Error updating status to success for deployment %d: %s\n", depId, err.Error())
		}
	}()
}
