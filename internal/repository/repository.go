package repository

// Repository describes a repository tracked by Git Workspace Manager.
type Repository struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Status contains the dynamic Git status of a repository.
type Status struct {
	Branch    string
	Dirty     bool
	Modified  int
	Added     int
	Deleted   int
	Untracked int

	Ahead  int
	Behind int
}

// RepositoryStatus combines a repository with its current status.
type RepositoryStatus struct {
	Repository Repository
	Status     Status
}
