package main

import (
	"fmt"

	"github.com/docker/docker/registry"
)

const (
	INDEXNAME = "docker.io"
)

type registrySession struct {
	registry.Session
	indexInfo *registry.IndexInfo
}

//return a registrySession associated with the repository contained in imageName
func newRegistrySession(userName, password string) (*registrySession, error) {
	//IndexInfo
	indexInfo := &registry.IndexInfo{
		Name:     INDEXNAME,
		Mirrors:  []string{},
		Secure:   true,
		Official: true,
	}

	endpoint, err := registry.NewEndpoint(indexInfo)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Index endpoint: %s\n", endpoint)

	authConfig := &registry.AuthConfig{Username: userName, Password: password}

	var metaHeaders map[string][]string

	session, err := registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), endpoint, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry session: %v", err)
	}

	return &registrySession{*session, indexInfo}, nil
}
