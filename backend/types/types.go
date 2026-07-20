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
	Id        int64  `json:"id"`
	UserId    int64  `json:"user_id"`
	Name      string `json:"name"`
	Status    string `json:"status"` // draft | building | failed | success
	Url       string `json:"url"`
	RepoUrl   string `json:"repo_url"`
	CreatedAt string `json:"created_at"`
}
