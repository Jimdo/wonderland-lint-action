package validator

import (
	"fmt"
	"net/url"
	"strings"
)

type MetaInformation struct {
}

func (v *MetaInformation) ValidateDocumentationURI(uri string) error {
	uri = strings.TrimSpace(uri)
	if len(uri) > 0 {
		u, err := url.Parse(uri)
		if err != nil {
			return fmt.Errorf("%q is not a valid documentation URI: %s", uri, err)
		}
		if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("documentation URI has an invalid scheme %q", u.Scheme)
		}
	}
	return nil
}
