package dlrootfs

import (
	"fmt"
	"io"
)

type PullingJob struct {
	Session *HubSession

	LayerId string

	LayerData io.ReadCloser
	LayerInfo []byte
	LayerSize int

	Err error
}

func NewPullingJob(session *HubSession, layerId string) *PullingJob {
	return &PullingJob{Session: session, LayerId: layerId}
}

func (job *PullingJob) Start() {
	fmt.Printf("\tPulling fs layer %v\n", truncateID(job.LayerId))
	endpoints := job.Session.RepoData.Endpoints
	tokens := job.Session.RepoData.Tokens

	for _, ep := range endpoints {
		job.LayerInfo, job.LayerSize, job.Err = job.Session.GetRemoteImageJSON(job.LayerId, ep, tokens)
		if job.Err != nil {
			continue
		}
		job.LayerData, job.Err = job.Session.GetRemoteImageLayer(job.LayerId, ep, tokens, int64(job.LayerSize))
	}

	fmt.Printf("\tDone %v\n", truncateID(job.LayerId))
}

func (job *PullingJob) Error() error {
	return job.Err
}

func (job *PullingJob) ID() string {
	return job.LayerId
}
