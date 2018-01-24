package ecsmetadata

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func TestECSMetadataService_GetService_Success(t *testing.T) {
	cluster := "foo"
	name := "bar"
	expected := &ecs.Service{
		ServiceArn: aws.String("arn:..."),
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == fmt.Sprintf("/v1/cluster/%s/service/%s", cluster, name) {
			if err := json.NewEncoder(w).Encode(expected); err != nil {
				t.Fatalf("Could not send JSON response in test server: %s", err)
			}
		}
	}))
	s := NewECSMetadataService(Config{
		BaseURI: ts.URL,
	})

	actual, err := s.GetService(cluster, name)
	if err != nil {
		t.Errorf("should not return an error when everything went well (err: %s)", err)
	}
	if aws.StringValue(actual.ServiceArn) != aws.StringValue(expected.ServiceArn) {
		t.Errorf("should return the same data as returned by the server. (%#v != %#v)", expected, actual)
	}
}

func TestECSMetadataService_GetService_NotFound(t *testing.T) {
	cluster := "foo"
	name := "test-service"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == fmt.Sprintf("/v1/cluster/%s/service/%s", cluster, name) {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	s := NewECSMetadataService(Config{
		BaseURI: ts.URL,
	})

	actual, err := s.GetService(cluster, name)
	if err != nil {
		t.Errorf("should not return an error when a resource was not found (err: %s)", err)
	}
	if actual != nil {
		t.Errorf("should not return a result when the resource was not found (%#v)", actual)
	}
}

func TestECSMetadataService_GetService_ServerError(t *testing.T) {
	cluster := "foo"
	name := "test-service"

	retryCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == fmt.Sprintf("/v1/cluster/%s/service/%s", cluster, name) {
			retryCount++
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "{\"error\": \"FooBar\"}")
		}
	}))
	s := NewECSMetadataService(Config{
		BaseURI: ts.URL,
	})

	actual, err := s.GetService(cluster, name)
	if err == nil {
		t.Error("should return an error when a server error occurred")
	}
	if actual != nil {
		t.Errorf("should not return a result when a server error occurred (%#v)", err)
	}
	if retryCount <= 1 {
		t.Error("should retry when a server error occurred")
	}
}
