// +build integration

package validator

import (
	"testing"

	"github.com/Jimdo/wonderland-validator/docker/registry"
)

func TestValidateImage_Valid(t *testing.T) {
	valid := []string{
		"nginx",
		"debian:jessie",
	}
	for _, image := range valid {
		v := &DockerImage{
			DockerImageService: registry.NewImageService(nil),
		}
		if err := v.validateImage(image); err != nil {
			t.Errorf("image %s should exist (err: %s)", image, err)
		}
	}
}

func TestValidateImage_Invalid(t *testing.T) {
	invalid := []string{
		"xyz",
		"debian:foo",
	}
	for _, image := range invalid {
		v := &DockerImage{
			DockerImageService: registry.NewImageService(nil),
		}
		if err := v.validateImage(image); err == nil {
			t.Errorf("image %s should not exist", image)
		}
	}
}
