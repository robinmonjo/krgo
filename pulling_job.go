package main

import (
	"fmt"
	"io"

	"github.com/docker/docker/registry"
)

type PullingJob struct {
	Session  *registrySession
	RepoData *registry.RepositoryData

	LayerId string

	LayerData io.ReadCloser
	LayerInfo []byte
	LayerSize int

	Err error
}

func NewPullingJob(session *registrySession, repoData *registry.RepositoryData, layerId string) *PullingJob {
	return &PullingJob{Session: session, RepoData: repoData, LayerId: layerId}
}

func (job *PullingJob) Start() {
	fmt.Printf("\t%v\n", job.LayerId)
	endpoints := job.RepoData.Endpoints
	tokens := job.RepoData.Tokens

	for _, ep := range endpoints {
		job.LayerInfo, job.LayerSize, job.Err = job.Session.GetRemoteImageJSON(job.LayerId, ep, tokens)
		if job.Err != nil {
			continue
		}
		job.LayerData, job.Err = job.Session.GetRemoteImageLayer(job.LayerId, ep, tokens, int64(job.LayerSize))
	}

	fmt.Printf("\tDone %v\n", job.LayerId)
}

func (job *PullingJob) Error() error {
	return job.Err
}

func (job *PullingJob) ID() string {
	return job.LayerId
}
