package dlrootfs

import (
	"log"
	"sync"
)

type Job interface {
	Start()
	Error() error
	ID() string
}

type Queue struct {
	Concurrency   int
	NbRunningJob  int
	WaitingJobs   []Job
	Lock          *sync.Mutex
	DoneChan      chan bool
	PerJobChan    chan string
	CompletedJobs map[string]Job
}

func NewQueue(concurrency int) *Queue {
	doneChan := make(chan bool)
	perJobChan := make(chan string, 10000)
	return &Queue{Concurrency: concurrency, Lock: &sync.Mutex{}, DoneChan: doneChan, PerJobChan: perJobChan, CompletedJobs: make(map[string]Job)}
}

func (queue *Queue) Enqueue(job Job) {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()

	if !queue.canLaunchJob() {
		//concurrency limit reached, make the job wait
		queue.WaitingJobs = append(queue.WaitingJobs, job)
		return
	}
	queue.startJob(job)
}

func (queue *Queue) startJob(job Job) {
	queue.NbRunningJob++
	go func() {
		//start the job
		job.Start()
		queue.dequeue(job)
	}()
}

func (queue *Queue) dequeue(job Job) {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()

	if job.Error() != nil {
		log.Fatal(job.Error())
	}
	queue.CompletedJobs[job.ID()] = job
	queue.PerJobChan <- job.ID()

	queue.NbRunningJob--
	if queue.canLaunchJob() && len(queue.WaitingJobs) > 0 {
		queue.startJob(queue.WaitingJobs[0])
		queue.WaitingJobs = append(queue.WaitingJobs[:0], queue.WaitingJobs[1:]...)
	}
	if len(queue.WaitingJobs) == 0 && queue.NbRunningJob == 0 {
		queue.DoneChan <- true
	}
}

func (queue *Queue) canLaunchJob() bool {
	return queue.NbRunningJob < queue.Concurrency
}

func (queue *Queue) CompletedJobWithID(jobId string) Job {
	return queue.CompletedJobs[jobId]
}
