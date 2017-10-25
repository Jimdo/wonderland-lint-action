package validator

import (
	"strings"
	"testing"
)

func TestValidateImageName_Valid(t *testing.T) {
	valid := []string{
		"nginx",
		"nginx:1.5",
		"jimdo/nginx",
		"jimdo/nginx:1.5",
		"registry.jimdo-platform.net/nginx",
		"registry.jimdo-platform.net/fast-nginx",
		"registry.jimdo-platform.net/nginx:1.5",
		"quay.io/jimdo_wonderland_stage/wonderland-deployer:master",
		"quay.io/jimdo_wonderland_prod/wonderland-deployer:foo",
		strings.Repeat("a", 255),
	}
	for _, image := range valid {
		v := &DockerImage{}
		if err := v.validateImageName(image); err != nil {
			t.Errorf("%s should be a valid image name (err: %s)", image, err)
		}
	}
}

func TestValidateImageName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"https://registry.jimdo-platform.net/nginx",
		strings.Repeat("a", 256),
		"registry.jimdo-platform.net/",
		"registry.jimdo-platform.net/nginx?",
		"registry.jimdo-platform.net/nginx:^^",
		"registry.jimdo-platform.net/NGINX",
		"registry.jimdo-platform.net/6677fb3347146c745af5b0863f8b4417bac0f4a8e58ab8ee96e7f68693a12d1d",
		"6677fb3347146c745af5b0863f8b4417bac0f4a8e58ab8ee96e7f68693a12d1d",
		"scratch",
	}
	for _, image := range invalid {
		v := &DockerImage{}
		if err := v.validateImageName(image); err == nil {
			t.Errorf("%s should not be a valid image name", image)
		}
	}
}
