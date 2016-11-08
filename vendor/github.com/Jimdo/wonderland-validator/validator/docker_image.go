package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Jimdo/wonderland-validator/docker/registry"
)

const (
	imageRepositoryMaxLength = 255
)

var (
	imageNameRegexp = regexp.MustCompile(`^[-_.:/a-z0-9]+$`)
	imageIDRegexp   = regexp.MustCompile(`^([a-f0-9]{64})$`)
)

type DockerImage struct {
	DockerImageService *registry.ImageService
}

func (v *DockerImage) Validate(image string) error {
	if err := v.validateImageName(image); err != nil {
		return err
	}
	if err := v.validateImage(image); err != nil {
		return fmt.Errorf("could not validate existence of Docker image %q: %s", image, err)
	}
	return nil
}

func (v *DockerImage) validateImageName(name string) error {
	if name == "" {
		return fmt.Errorf("image name is missing")
	}
	if strings.Contains(name, "://") {
		return fmt.Errorf("image name must not contain a schema")
	}
	if !imageNameRegexp.MatchString(name) {
		return fmt.Errorf("image name contains invalid characters")
	}
	_, repository := v.splitImageName(name)
	if repository == "" {
		return fmt.Errorf("image repository name must not be empty")
	}
	if repository == "scratch" {
		return fmt.Errorf("image repository name must not be 'scratch'")
	}
	if len(repository) > imageRepositoryMaxLength {
		return fmt.Errorf("image repository name too long")
	}
	if !strings.Contains(repository, "/") {
		if imageIDRegexp.MatchString(repository) {
			return fmt.Errorf("image repository name must not be an image ID")
		}
	}
	return nil
}

func (v *DockerImage) validateImage(imageName string) error {
	registryName, repoName := v.splitImageName(imageName)
	tagName := "latest"

	// TODO: move this to v.splitImageName
	nameParts := strings.SplitN(repoName, ":", 2)
	if len(nameParts) == 2 {
		repoName = nameParts[0]
		tagName = nameParts[1]
	}

	r, err := v.DockerImageService.GetRepository(registryName, repoName)
	if err != nil {
		return err
	}

	exists, err := r.HasTag(tagName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("image %s does not exist", imageName)
	}
	return nil
}

// Based on https://github.com/docker/docker/blob/c9208953fac6174bb205bd1b3705f81a602869c2/registry/config.go#L286-L301
func (v *DockerImage) splitImageName(name string) (string, string) {
	nameParts := strings.SplitN(name, "/", 2)
	var registry, repository string
	if len(nameParts) == 1 || (!strings.Contains(nameParts[0], ".") &&
		!strings.Contains(nameParts[0], ":") && nameParts[0] != "localhost") {
		// FIXME: return docker.io as default
		registry = ""
		repository = name
	} else {
		registry = nameParts[0]
		repository = nameParts[1]
	}
	return registry, repository
}
