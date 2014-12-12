package dlrootfs

import (
	"fmt"
	"io"

	"github.com/docker/docker/registry"
)

type PullingJob struct {
	Session        *registry.Session
	RepositoryData *registry.RepositoryData

	LayerId string

	LayerData io.ReadCloser
	LayerInfo []byte
	LayerSize int

	Err error
}

func NewPullingJob(session *registry.Session, repoData *registry.RepositoryData, layerId string) *PullingJob {
	return &PullingJob{Session: session, RepositoryData: repoData, LayerId: layerId}
}

func (job *PullingJob) Start() {
	fmt.Printf("\tPulling fs layer %v\n", truncateID(job.LayerId))
	endpoint := job.RepositoryData.Endpoints[0]
	tokens := job.RepositoryData.Tokens

	job.LayerInfo, job.LayerSize, job.Err = job.Session.GetRemoteImageJSON(job.LayerId, endpoint, tokens)
	if job.Err != nil {
		return
	}
	job.LayerData, job.Err = job.Session.GetRemoteImageLayer(job.LayerId, endpoint, tokens, int64(job.LayerSize))
	fmt.Printf("\tDone %v\n", truncateID(job.LayerId))
}

func (job *PullingJob) Error() error {
	return job.Err
}

func (job *PullingJob) ID() string {
	return job.LayerId
}
