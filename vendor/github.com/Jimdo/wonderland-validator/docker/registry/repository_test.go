// +build integration

package registry_test

import (
	"os"
	"testing"

	"github.com/Jimdo/wonderland-validator/docker/registry"
)

func TestRepositoryHasTag(t *testing.T) {
	tests := []struct {
		registryName   string
		repositoryName string
		tagName        string
		exists         bool
	}{
		{"", "nginx", "latest", true},
		{"", "debian", "jessie", true},
		{"", "debian", "foo", false},
		{"", "elcolio/etcd", "2.0.10", true},
		/*
			quay.io test currently fails with
			repository_test.go:44: Get /v2/coreos/etcd/tags/list?next_page=gAAAAABZGZsClv3-MFbisQRnC4tst516qdzAoiIlIIkY60_gL-LP3I0Qlq_sI3Hkq2ucRxtzAO97XaEnGFKdZMFieDNs7UJivg%3D%3D&n=50: unsupported protocol scheme ""
			ignoring this for now as we don't know when it started, what it is and it is (hopefully) not critical
		*/
		//		{"quay.io", "coreos/etcd", "v2.2.1", true},
		{"registry.jimdo-platform.net", "auth-proxy", "latest", true},
		{"registry.jimdo-platform.net", "xyz", "latest", false},
	}

	wonderlandRegistryUser := os.Getenv("WONDERLAND_USER")
	wonderlandRegistryPass := os.Getenv("WONDERLAND_PASS")

	imageService := registry.NewImageService([]registry.Credential{{
		Username: wonderlandRegistryUser,
		Password: wonderlandRegistryPass,
		Host:     "registry.jimdo-platform.net",
	}})
	for _, test := range tests {
		r, err := imageService.GetRepository(test.registryName, test.repositoryName)
		if err != nil {
			t.Error(err)
			continue
		}
		exists, err := r.HasTag(test.tagName)
		if err != nil {
			t.Error(err)
		}
		if test.exists != exists {
			t.Errorf("Test failed for %s:%s in registry %s\n",
				test.repositoryName, test.tagName, test.registryName)
		}
	}
}
