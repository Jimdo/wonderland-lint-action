package registry

import "net/url"

func newCredentialStore(credentials []Credential) *CredentialStore {
	credentialsMap := map[string]Credential{}
	for _, credential := range credentials {
		credentialsMap[credential.Host] = credential
	}

	return &CredentialStore{
		credentials: credentialsMap,
	}
}

type CredentialStore struct {
	credentials map[string]Credential
}

type Credential struct {
	Username string
	Password string
	Host     string
}

func (s *CredentialStore) Basic(u *url.URL) (string, string) {
	credential, ok := s.credentials[u.Host]
	if !ok {
		return "", ""
	}

	return credential.Username, credential.Password
}

func (s *CredentialStore) RefreshToken(u *url.URL, service string) string {
	return ""
}

func (s *CredentialStore) SetRefreshToken(u *url.URL, service, token string) {
}
