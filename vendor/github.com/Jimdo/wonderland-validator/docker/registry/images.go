package registry

import "github.com/docker/distribution/registry/client/auth"

func NewImageService(credentials []Credential) *ImageService {
	credentialStore := newCredentialStore(credentials)

	return &ImageService{
		credentialStore: credentialStore,
	}
}

type ImageService struct {
	credentialStore auth.CredentialStore
}

func (s *ImageService) GetRepository(registryName, repoName string) (*Repository, error) {
	return NewRepository(registryName, repoName, s.credentialStore)
}
