package types

type User struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	RepoUrl  string `json:"repo_url"`
}

type JwtPayload struct {
	Id    int64
	Email string
}

type Deployment struct {
	Id         int64  `json:"id"`
	UserId     int64  `json:"user_id"`
	Name       string `json:"name"`
	Status     string `json:"status"` // queued | pending | deployed | failed
	Url        string `json:"url"`
	RepoUrl    string `json:"repo_url"`
	Stack      string `json:"stack"`
	InstallCmd string `json:"install_cmd"`
	BuildCmd   string `json:"build_cmd"`
	RunCmd     string `json:"run_cmd"`
	CreatedAt  string `json:"created_at"`
}
