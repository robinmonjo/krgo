package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/registry"
)

type PullingV2Job struct {
	Session   *registrySession
	Endpoint  *registry.Endpoint
	Auth      *registry.RequestAuthorization
	ImageName string
	SumStr    string

	LayerId string

	LayerDataReader   io.ReadCloser
	LayerTarSumReader tarsum.TarSum
	LayerSize         int64

	Err error
}

func NewPullingV2Job(session *registrySession, endpoint *registry.Endpoint, auth *registry.RequestAuthorization, imageName, sumStr string) *PullingV2Job {
	return &PullingV2Job{Session: session, Endpoint: endpoint, Auth: auth, ImageName: imageName, SumStr: sumStr}
}

func (job *PullingV2Job) Start() {
	chunks := strings.SplitN(job.SumStr, ":", 2)
	if len(chunks) < 2 {
		job.Err = fmt.Errorf("expected 2 parts in the sumStr, got %#v", chunks)
		return
	}
	sumType, checksum := chunks[0], chunks[1]
	fmt.Printf("\t%s ...\n", checksum)
	job.LayerDataReader, job.LayerSize, job.Err = job.Session.GetV2ImageBlobReader(job.Endpoint, job.ImageName, sumType, checksum, job.Auth)
	if job.Err != nil {
		return
	}

	job.LayerTarSumReader, job.Err = tarsum.NewTarSumForLabel(job.LayerDataReader, true, sumType)
	if job.Err != nil {
		return
	}

	fmt.Printf("\tDone %s\n", checksum)
}

func (job *PullingV2Job) Error() error {
	return job.Err
}

func (job *PullingV2Job) ID() string {
	return job.SumStr
}
