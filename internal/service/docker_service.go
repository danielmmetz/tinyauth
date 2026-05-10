package service

import (
	"context"
	"strings"

	"github.com/steveiliop56/tinyauth/internal/config"
	"github.com/steveiliop56/tinyauth/internal/utils/decoders"
	"github.com/steveiliop56/tinyauth/internal/utils/tlog"

	container "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerService struct {
	client      *client.Client
	context     context.Context
	isConnected bool
}

func NewDockerService() *DockerService {
	return &DockerService{}
}

func (docker *DockerService) Init() error {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	ctx := context.Background()
	client.NegotiateAPIVersion(ctx)

	docker.client = client
	docker.context = ctx

	_, err = docker.client.Ping(docker.context)

	if err != nil {
		tlog.App.Debug().Err(err).Msg("Docker not connected")
		docker.isConnected = false
		docker.client = nil
		docker.context = nil
		return nil
	}

	docker.isConnected = true
	tlog.App.Debug().Msg("Docker connected")

	return nil
}

func (docker *DockerService) getContainers() ([]container.Summary, error) {
	containers, err := docker.client.ContainerList(docker.context, container.ListOptions{})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (docker *DockerService) inspectContainer(containerId string) (container.InspectResponse, error) {
	inspect, err := docker.client.ContainerInspect(docker.context, containerId)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return inspect, nil
}

func (docker *DockerService) GetLabels(appDomain string) (config.App, error) {
	if !docker.isConnected {
		tlog.App.Debug().Msg("Docker not connected, returning empty labels")
		return config.App{}, nil
	}

	containers, err := docker.getContainers()
	if err != nil {
		return config.App{}, err
	}

	for _, ctr := range containers {
		inspect, err := docker.inspectContainer(ctr.ID)
		if err != nil {
			return config.App{}, err
		}

		labels, err := decoders.DecodeLabels[config.Apps](inspect.Config.Labels, "apps")
		if err != nil {
			return config.App{}, err
		}

		for appName, appLabels := range labels.Apps {
			if appLabels.Config.Domain == appDomain {
				tlog.App.Debug().Str("id", inspect.ID).Str("name", inspect.Name).Msg("Found matching container by domain")
				return appLabels, nil
			}

			if strings.SplitN(appDomain, ".", 2)[0] == appName {
				tlog.App.Debug().Str("id", inspect.ID).Str("name", inspect.Name).Msg("Found matching container by app name")
				return appLabels, nil
			}
		}
	}

	tlog.App.Debug().Msg("No matching container found, returning empty labels")
	return config.App{}, nil
}

func (docker *DockerService) GetDomains() ([]string, error) {
	domains := []string{"*"}

	if !docker.isConnected {
		return domains, nil
	}

	containers, err := docker.getContainers()
	if err != nil {
		tlog.App.Warn().Err(err).Msg("Failed to get Docker containers for domain enumeration")
		return domains, nil
	}

	seenDomains := map[string]bool{"*": true}

	for _, ctr := range containers {
		inspect, err := docker.inspectContainer(ctr.ID)
		if err != nil {
			tlog.App.Debug().Err(err).Str("container_id", ctr.ID).Msg("Failed to inspect container")
			continue
		}

		labels, err := decoders.DecodeLabels[config.Apps](inspect.Config.Labels, "apps")
		if err != nil {
			tlog.App.Debug().Err(err).Str("container_id", ctr.ID).Msg("Failed to decode container labels")
			continue
		}

		for _, appLabels := range labels.Apps {
			domain := appLabels.Config.Domain
			if domain != "" && !seenDomains[domain] {
				domains = append(domains, domain)
				seenDomains[domain] = true
			}
		}
	}

	return domains, nil
}
