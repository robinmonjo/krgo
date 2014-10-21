package main

import (
	"fmt"
	"io"

	"github.com/docker/docker/registry"
)

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
	fmt.Printf("\tDownloading layer %v\n", job.LayerId)
	endpoint := job.RepositoryData.Endpoints[0]
	tokens := job.RepositoryData.Tokens

	job.LayerInfo, job.LayerSize, job.Err = job.Session.GetRemoteImageJSON(job.LayerId, endpoint, tokens)
	if job.Err != nil {
		return
	}
	job.LayerData, job.Err = job.Session.GetRemoteImageLayer(job.LayerId, endpoint, tokens, int64(job.LayerSize))
	fmt.Printf("\tDone %v\n", job.LayerId)
}

func (job *DownloadJob) Error() error {
	return job.Err
}

func (job *DownloadJob) ID() string {
	return job.LayerId
}
