package queue

import (
	"sync"
	"testing"
	"time"
)

func TestEnqueueAndHandle(t *testing.T) {
	q := &Queue{jobs: make(chan Job, 4)}

	var mu sync.Mutex
	var got []int64
	done := make(chan struct{}, 3)

	q.Start(2, func(job Job) {
		mu.Lock()
		got = append(got, job.DeploymentId)
		mu.Unlock()
		done <- struct{}{}
	})

	for _, id := range []int64{1, 2, 3} {
		if err := q.Enqueue(Job{DeploymentId: id}); err != nil {
			t.Fatalf("Enqueue(%d): %s", id, err)
		}
	}

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("a queued job was never handled")
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 {
		t.Errorf("handled %d jobs, want 3", len(got))
	}
}

// A full queue must be reported to the caller rather than blocking the HTTP handler that
// is enqueueing - that is what lets the API answer 503 and mark the deployment failed.
func TestEnqueueDoesNotBlockWhenFull(t *testing.T) {
	q := &Queue{jobs: make(chan Job, 2)}

	if err := q.Enqueue(Job{DeploymentId: 1}); err != nil {
		t.Fatalf("first enqueue: %s", err)
	}
	if err := q.Enqueue(Job{DeploymentId: 2}); err != nil {
		t.Fatalf("second enqueue: %s", err)
	}
	if q.Depth() != 2 {
		t.Errorf("Depth = %d, want 2", q.Depth())
	}

	returned := make(chan error, 1)
	go func() { returned <- q.Enqueue(Job{DeploymentId: 3}) }()

	select {
	case err := <-returned:
		if err == nil {
			t.Error("enqueueing onto a full queue should fail, not silently drop the job")
		}
	case <-time.After(time.Second):
		t.Fatal("Enqueue blocked on a full queue - an HTTP handler would hang here")
	}
}

// Start is wired up in main and must stay safe if that ever happens twice.
func TestStartIsIdempotent(t *testing.T) {
	q := &Queue{jobs: make(chan Job, 1)}

	var mu sync.Mutex
	calls := 0
	count := func(Job) {
		mu.Lock()
		calls++
		mu.Unlock()
	}

	q.Start(1, count)
	q.Start(1, count) // must not add a second pool

	if err := q.Enqueue(Job{DeploymentId: 1}); err != nil {
		t.Fatalf("Enqueue: %s", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := calls
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("job was handled %d times, want exactly 1", calls)
	}
}
