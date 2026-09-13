package workspace

import (
	"sync"

	"gitworkspacefun/internal/repository"
)

// Result is the outcome of processing one repository.
type Result struct {
	Repository repository.Repository
	Error      error
}

// ForEach processes repositories with a bounded number of workers. Results
// retain the same order as the input, while one failure does not stop others.
func ForEach(repositories []repository.Repository, workers int, fn func(repository.Repository) error) []Result {
	if workers < 1 {
		workers = 1
	}
	if workers > len(repositories) && len(repositories) > 0 {
		workers = len(repositories)
	}
	results := make([]Result, len(repositories))
	jobs := make(chan int)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				repo := repositories[index]
				results[index] = Result{Repository: repo, Error: fn(repo)}
			}
		}()
	}
	for index := range repositories {
		jobs <- index
	}
	close(jobs)
	group.Wait()
	return results
}
