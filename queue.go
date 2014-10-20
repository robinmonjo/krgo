package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/registry"
)

type Queue struct {
	Concurrency   int
	NbRunningJob  int
	WaitingJobs   []Job
	Lock          *sync.Mutex
	DoneChan      chan bool
	PerJobChan    chan string
	CompletedJobs []Job
}

func NewQueue(concurrency int) *Queue {
	doneChan := make(chan bool)
	perJobChan := make(chan string, 10000)
	return &Queue{Concurrency: concurrency, Lock: &sync.Mutex{}, DoneChan: doneChan, PerJobChan: perJobChan}
}

func (queue *Queue) enqueue(job Job) {
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

	assertErr(job.Error())
	queue.CompletedJobs = append(queue.CompletedJobs, job)
	queue.PerJobChan <- job.ID()

	queue.NbRunningJob--
	if queue.canLaunchJob() && len(queue.WaitingJobs) > 0 {
		queue.startJob(queue.WaitingJobs[0])
		queue.WaitingJobs = append(queue.WaitingJobs[:0], queue.WaitingJobs[1:]...) //remove first waiting job
	}
	if len(queue.WaitingJobs) == 0 && queue.NbRunningJob == 0 {
		queue.DoneChan <- true
	}
}

func (queue *Queue) canLaunchJob() bool {
	return queue.NbRunningJob < queue.Concurrency
}

func (queue *Queue) completedJobWithLayerId(layerId string) *DownloadJob {
	for _, job := range queue.CompletedJobs {
		if job.(*DownloadJob).LayerId == layerId {
			return job.(*DownloadJob)
		}
	}
	return nil
}

type Job interface {
	Start()
	Error() error
	ID() string
}

type DownloadJob struct {
	Session        *registry.Session
	RepositoryData *registry.RepositoryData

	LayerId string

	LayerData io.ReadCloser
	LayerInfo []byte
	LayerSize int

	Err error
}

func NewDownloadJob(session *registry.Session, repoData *registry.RepositoryData, layerId string) *DownloadJob {
	return &DownloadJob{Session: session, RepositoryData: repoData, LayerId: layerId}
}

func (job *DownloadJob) Start() {
	fmt.Printf("Starting download layer %v\n", job.LayerId)
	endpoint := job.RepositoryData.Endpoints[0]
	tokens := job.RepositoryData.Tokens

	job.LayerInfo, job.LayerSize, job.Err = job.Session.GetRemoteImageJSON(job.LayerId, endpoint, tokens)
	if job.Err != nil {
		return
	}
	job.LayerData, job.Err = job.Session.GetRemoteImageLayer(job.LayerId, endpoint, tokens, int64(job.LayerSize))
	fmt.Printf("Done download layer %v\n", job.LayerId)
}

func (job *DownloadJob) Error() error {
	return job.Err
}

func (job *DownloadJob) ID() string {
	return job.LayerId
}
