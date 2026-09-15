package queue

import (
	"fmt"
	"sync"
)

// Job is everything a worker needs to build and publish a deployment.
type Job struct {
	DeploymentId int64
	Stack        string
	RepoUrl      string
	InstallCmd   string
	BuildCmd     string
	RunCmd       string
}

// Handler runs a single job. It is expected to move the deployment out of the
// 'queued' state and to record its own failures.
type Handler func(Job)

// Queue is an in-process build-job queue served by a pool of workers. Postgres is the
// durable record: jobs left in the 'queued' state are re-enqueued on startup.
type Queue struct {
	jobs    chan Job
	handler Handler
	start   sync.Once
}

var defaultQueue = &Queue{jobs: make(chan Job, 128)}

// Default returns the process-wide build queue.
func Default() *Queue { return defaultQueue }

// Start launches the worker pool. Calling it more than once is a no-op.
func (q *Queue) Start(workers int, handler Handler) {
	q.start.Do(func() {
		q.handler = handler
		for i := 1; i <= workers; i++ {
			go q.work(i)
		}
		fmt.Printf("[Queue] started %d build worker(s)\n", workers)
	})
}

func (q *Queue) work(id int) {
	for job := range q.jobs {
		fmt.Printf("[Queue] worker %d picked up deployment %d\n", id, job.DeploymentId)
		q.handler(job)
	}
}

// Enqueue adds a job to the queue. It never blocks: a full queue is reported to the caller
// so the deployment can be marked failed instead of silently stalling the request.
func (q *Queue) Enqueue(job Job) error {
	select {
	case q.jobs <- job:
		fmt.Printf("[Queue] queued deployment %d (%s)\n", job.DeploymentId, job.Stack)
		return nil
	default:
		return fmt.Errorf("build queue is full, try again shortly")
	}
}

// Depth reports how many jobs are waiting to be picked up.
func (q *Queue) Depth() int { return len(q.jobs) }
