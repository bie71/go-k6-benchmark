package main

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// Job merepresentasikan unit pekerjaan yang dikirim ke worker pool.
type Job struct {
	ID   int
	Data any
}

// JobResult merepresentasikan hasil pekerjaan yang telah diselesaikan oleh worker.
type JobResult struct {
	JobID    int
	Duration time.Duration
	Data     any
	Err      error
}

// Pool mengelola pool worker berbasis goroutine, channel, dan context.
type Pool struct {
	workers int
	jobs    chan Job
	results chan JobResult
	wg      sync.WaitGroup
}

// NewPool menginisialisasi Pool dengan jumlah worker dan kapasitas antrean tertentu.
func NewPool(workers int, queueSize int) *Pool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if queueSize <= 0 {
		queueSize = workers * 2
	}
	return &Pool{
		workers: workers,
		jobs:    make(chan Job, queueSize),
		results: make(chan JobResult, queueSize),
	}
}

// Start menjalankan sejumlah goroutine worker untuk memproses pekerjaan dari channel jobs.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}

					// Simulasi beban komputasi/backend 20-50 milidetik
					sleepDuration := time.Duration(20+rand.Intn(31)) * time.Millisecond
					select {
					case <-ctx.Done():
						return
					case <-time.After(sleepDuration):
					}

					result := JobResult{
						JobID:    job.ID,
						Duration: sleepDuration,
						Data:     job.Data,
					}

					select {
					case <-ctx.Done():
						return
					case p.results <- result:
					}
				}
			}
		}()
	}
}

// Stop menutup channel jobs dan menunggu seluruh worker selesai sebelum menutup channel results.
func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// Process mendistribusikan sekumpulan job ke worker pool dan mengembalikan seluruh hasil eksekusi.
func (p *Pool) Process(ctx context.Context, jobList []Job) []JobResult {
	p.Start(ctx)

	go func() {
		defer close(p.jobs)
		for _, j := range jobList {
			select {
			case <-ctx.Done():
				return
			case p.jobs <- j:
			}
		}
	}()

	go func() {
		p.wg.Wait()
		close(p.results)
	}()

	var output []JobResult
	for res := range p.results {
		output = append(output, res)
	}

	return output
}
