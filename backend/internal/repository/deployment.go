package repository

import (
	"fmt"

	"github.com/sudhanshu042004/orcs/database"
	"github.com/sudhanshu042004/orcs/types"
)

func CreateDeployment(userId int64, name string, repoUrl string, status string) (int64, error) {
	var deploymentId int64
	q := `INSERT INTO deployments(user_id, name, repo_url, status, url) VALUES($1, $2, $3, $4, '') RETURNING id`
	err := database.DB.QueryRow(q, userId, name, repoUrl, status).Scan(&deploymentId)
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

func GetDeployments(userId int64) ([]types.Deployment, error) {
	q := `SELECT id, user_id, name, status, url, repo_url, created_at FROM deployments WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := database.DB.Query(q, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deployments := []types.Deployment{}
	for rows.Next() {
		var d types.Deployment
		var t interface{}
		err := rows.Scan(&d.Id, &d.UserId, &d.Name, &d.Status, &d.Url, &d.RepoUrl, &t)
		if err != nil {
			return nil, err
		}
		if t != nil {
			d.CreatedAt = fmt.Sprintf("%v", t)
		}
		deployments = append(deployments, d)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return deployments, nil
}

func GetDeployment(id int64, userId int64) (types.Deployment, error) {
	var d types.Deployment
	var t interface{}
	q := `SELECT id, user_id, name, status, url, repo_url, created_at FROM deployments WHERE id = $1 AND user_id = $2`
	err := database.DB.QueryRow(q, id, userId).Scan(&d.Id, &d.UserId, &d.Name, &d.Status, &d.Url, &d.RepoUrl, &t)
	if err != nil {
		return types.Deployment{}, err
	}
	if t != nil {
		d.CreatedAt = fmt.Sprintf("%v", t)
	}
	return d, nil
}

func DeleteDeployment(id int64, userId int64) error {
	q := `DELETE FROM deployments WHERE id = $1 AND user_id = $2`
	_, err := database.DB.Exec(q, id, userId)
	return err
}
