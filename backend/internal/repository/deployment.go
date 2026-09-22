package repository

import (
	"database/sql"
	"fmt"

	"github.com/sudhanshu042004/orcs/database"
	"github.com/sudhanshu042004/orcs/types"
)

const deploymentColumns = `id, user_id, name, status, url, repo_url, stack, install_cmd, build_cmd, run_cmd, created_at`

// CreateDeployment records a new deployment. It is always created in the 'queued' state -
// the build itself is picked up later by a worker.
func CreateDeployment(userId int64, d types.Deployment) (int64, error) {
	var deploymentId int64
	q := `INSERT INTO deployments(user_id, name, repo_url, status, url, stack, install_cmd, build_cmd, run_cmd)
	      VALUES($1, $2, $3, 'queued', '', $4, $5, $6, $7) RETURNING id`
	err := database.DB.QueryRow(q, userId, d.Name, d.RepoUrl, d.Stack, d.InstallCmd, d.BuildCmd, d.RunCmd).Scan(&deploymentId)
	if err != nil {
		fmt.Printf("error while creating deployment %s\n", err.Error())
		return 0, err
	}
	return deploymentId, nil
}

func UpdateDeploymentStatus(id int64, status string, url string) error {
	q := `UPDATE deployments SET status = $1, url = $2 WHERE id = $3`
	_, err := database.DB.Exec(q, status, url, id)
	if err != nil {
		fmt.Printf("error while updating deployment %s\n", err.Error())
		return err
	}
	return nil
}

func scanDeployment(scan func(dest ...any) error) (types.Deployment, error) {
	var d types.Deployment
	var t interface{}
	err := scan(&d.Id, &d.UserId, &d.Name, &d.Status, &d.Url, &d.RepoUrl, &d.Stack, &d.InstallCmd, &d.BuildCmd, &d.RunCmd, &t)
	if err != nil {
		return types.Deployment{}, err
	}
	if t != nil {
		d.CreatedAt = fmt.Sprintf("%v", t)
	}
	return d, nil
}

func GetDeployments(userId int64) ([]types.Deployment, error) {
	q := `SELECT ` + deploymentColumns + ` FROM deployments WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := database.DB.Query(q, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deployments := []types.Deployment{}
	for rows.Next() {
		d, err := scanDeployment(rows.Scan)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return deployments, nil
}

func GetDeployment(id int64, userId int64) (types.Deployment, error) {
	q := `SELECT ` + deploymentColumns + ` FROM deployments WHERE id = $1 AND user_id = $2`
	return scanDeployment(database.DB.QueryRow(q, id, userId).Scan)
}

// GetQueuedDeployments returns every deployment still waiting for a worker, oldest first.
// Used on startup to put jobs back on the queue after a restart.
func GetQueuedDeployments() ([]types.Deployment, error) {
	q := `SELECT ` + deploymentColumns + ` FROM deployments WHERE status = 'queued' ORDER BY created_at ASC`
	rows, err := database.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deployments := []types.Deployment{}
	for rows.Next() {
		d, err := scanDeployment(rows.Scan)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

func DeleteDeployment(id int64, userId int64) error {
	q := `DELETE FROM deployments WHERE id = $1 AND user_id = $2`
	_, err := database.DB.Exec(q, id, userId)
	return err
}

func SaveDeploymentContainer(depId int64, containerId string) error {
	q := `INSERT INTO deployment_containers(deployment_id, container_id) VALUES($1, $2)`
	_, err := database.DB.Exec(q, depId, containerId)
	return err
}

// SetDeploymentHostPort records the host port a running app's container was published on.
func SetDeploymentHostPort(depId int64, hostPort int) error {
	q := `UPDATE deployment_containers SET host_port = $1 WHERE deployment_id = $2`
	_, err := database.DB.Exec(q, hostPort, depId)
	return err
}

// GetDeploymentHostPort returns the host port a running deployment answers on. A zero port
// means the deployment has no running container.
func GetDeploymentHostPort(depId int64) (int, error) {
	var hostPort sql.NullInt64
	q := `SELECT host_port FROM deployment_containers WHERE deployment_id = $1`
	if err := database.DB.QueryRow(q, depId).Scan(&hostPort); err != nil {
		return 0, err
	}
	return int(hostPort.Int64), nil
}

// GetPublicDeployment looks a deployment up without scoping it to an owner. Deployed sites
// are served to anyone who has the URL, so serving one cannot ask who is asking.
func GetPublicDeployment(id int64) (types.Deployment, error) {
	q := `SELECT ` + deploymentColumns + ` FROM deployments WHERE id = $1`
	return scanDeployment(database.DB.QueryRow(q, id).Scan)
}

func GetDeploymentContainer(depId int64) (string, error) {
	var containerId string
	q := `SELECT container_id FROM deployment_containers WHERE deployment_id = $1`
	err := database.DB.QueryRow(q, depId).Scan(&containerId)
	return containerId, err
}

func DeleteDeploymentContainer(depId int64) error {
	q := `DELETE FROM deployment_containers WHERE deployment_id = $1`
	_, err := database.DB.Exec(q, depId)
	return err
}
