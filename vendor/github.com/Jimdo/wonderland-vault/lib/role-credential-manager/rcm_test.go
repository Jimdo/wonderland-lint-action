package rcm

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	testVaultAddress            = "https://vault.example.com"
	testVaultRoleID             = "abcdefg12345"
	testVaultToken              = "12345abcdefg"
	testVaultTokenLeaseDuration = 1

	testAWSIAMRole                  = "rcm-library-test"
	testAWSAccessKeyID              = "AKIAABCDEFFG123456"
	testAWSSecretAccessKey          = "1234567abcdefgh"
	testAWSSessionToken             = "foo-bar"
	testAWSCredentialsLeaseDuration = 2

	vaultURIPathVersionPrefix = "/v1/"
)

var (
	mockVaultURIPathApproleLogin   = vaultURIPathVersionPrefix + vaultURIPathApproleLogin
	mockVaultURIPathRenewToken     = vaultURIPathVersionPrefix + vaultURIPathRenewToken
	mockVaultURIPathSTSCredentials = vaultURIPathVersionPrefix + vaultURIPathSTSCredentials + testAWSIAMRole
)

// vaultRequestsStore is used to record requests made to Vault in a mock
// first dimension is "by HTTP method", second is "by URL Path", so
// map["PUT"]["/foo/bar"] == 3 means that ther have been 3 requests like "PUT /foo/bar"
type vaultRequests map[string]map[string]int

func TestNew_InvalidConfig_NoVaultAddress(t *testing.T) {
	rcm, err := New("", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config: no Vault address")
	assert.Nil(t, rcm)
}

func TestNew_InvalidConfig_NoRoleID(t *testing.T) {
	rcm, err := New(testVaultAddress, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config: no Vault role ID")
	assert.Nil(t, rcm)
}

func TestNew_InvalidConfig_NoAWSIAMRole(t *testing.T) {
	rcm, err := New(
		testVaultAddress,
		testVaultRoleID,
		WithAWSClientConfigs(&aws.Config{}),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config: no AWS IAM role")
	assert.Nil(t, rcm)
}

func TestRoleCredentialManagerRun_InitializesVaultClientAndRenewsVaultToken(t *testing.T) {
	vaultRequests, ts := mockVault(t)

	rcm, err := New(
		ts.URL,
		testVaultRoleID,
		WithLogger(logrus.StandardLogger()),
	)
	assert.NoError(t, err)

	err = rcm.Init()
	assert.NoError(t, err)
	assert.NotNil(t, rcm.VaultClient)
	assert.Equal(t, rcm.VaultClient.Token(), testVaultToken)
	assert.NotZero(t, vaultRequests["PUT"][mockVaultURIPathRenewToken], "Run should have renewed the Vault token but didn't")
}

func TestRoleCredentialManagerRun_UpdatesAWSCredentials(t *testing.T) {
	vaultRequests, ts := mockVault(t)

	config := &aws.Config{}

	rcm, err := New(
		ts.URL,
		testVaultRoleID,
		WithAWSIAMRole(testAWSIAMRole),
		WithAWSClientConfigs(config),
	)
	assert.NoError(t, err)

	err = rcm.Init()
	assert.NoError(t, err)
	assert.NotNil(t, config.Credentials)

	creds, err := config.Credentials.Get()
	assert.NoError(t, err)
	assert.Equal(t, creds.AccessKeyID, testAWSAccessKeyID)
	assert.Equal(t, creds.SecretAccessKey, testAWSSecretAccessKey)
	assert.Equal(t, creds.SessionToken, testAWSSessionToken)
	assert.NotZero(t, vaultRequests["GET"][mockVaultURIPathSTSCredentials], "Run should have fetched AWS credentials but didn't")
}

func TestRoleCredentialManagerRun_InitializesVaultClientAndRenewsVaultTokenLazy(t *testing.T) {
	vaultRequests, ts := mockVault(t)

	rcm, err := New(
		ts.URL,
		testVaultRoleID,
		WithLogger(logrus.StandardLogger()),
	)
	assert.NoError(t, err)

	stop := make(chan interface{})

	go func() {
		err := rcm.Run(stop)
		assert.NoError(t, err)
	}()

	time.Sleep(1500 * time.Millisecond)
	close(stop)

	err = rcm.Init()
	assert.NoError(t, err)
	assert.NotNil(t, rcm.VaultClient)
	assert.Equal(t, rcm.VaultClient.Token(), testVaultToken)
	assert.NotZero(t, vaultRequests["PUT"][mockVaultURIPathRenewToken], "Run should have renewed the Vault token but didn't")
}

func TestRoleCredentialManagerRun_UpdatesAWSCredentialsLazy(t *testing.T) {
	vaultRequests, ts := mockVault(t)

	config := &aws.Config{}

	rcm, err := New(
		ts.URL,
		testVaultRoleID,
		WithAWSIAMRole(testAWSIAMRole),
		WithAWSClientConfigs(config),
	)
	assert.NoError(t, err)

	stop := make(chan interface{})

	go func() {
		err := rcm.Run(stop)
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)
	close(stop)

	creds, err := config.Credentials.Get()
	assert.NoError(t, err)
	assert.Equal(t, creds.AccessKeyID, testAWSAccessKeyID)
	assert.Equal(t, creds.SecretAccessKey, testAWSSecretAccessKey)
	assert.Equal(t, creds.SessionToken, testAWSSessionToken)
	assert.NotZero(t, vaultRequests["GET"][mockVaultURIPathSTSCredentials], "Run should have fetched AWS credentials but didn't")
}

func TestRoleCredentialManagerRun_ReturnsErrorsWhenVaultIsUnavailable(t *testing.T) {
	ts := mockVaultBrokenSTS(t)

	rcm, err := New(
		ts.URL,
		testVaultRoleID,
		WithAWSIAMRole(testAWSIAMRole),
		WithAWSClientConfigs(&aws.Config{}),
	)
	assert.NoError(t, err)

	err = rcm.Init()
	assert.Error(t, err)
}

func TestRoleCredentialManagerRun_IgnoreErrors(t *testing.T) {
	ts := mockVaultBrokenSTS(t)

	rcm, err := New(
		ts.URL,
		testVaultRoleID,
		WithIgnoreErrors(),
	)
	assert.NoError(t, err)

	stop := make(chan interface{})
	go func() {
		err := rcm.Run(stop)
		assert.NoError(t, err)
	}()

	time.Sleep(300 * time.Millisecond)
	close(stop)
}

func mockVault(t *testing.T) (vaultRequests, *httptest.Server) {
	t.Helper()

	requests := vaultRequests{
		"GET": map[string]int{
			mockVaultURIPathSTSCredentials: 0,
		},
		"PUT": map[string]int{
			mockVaultURIPathApproleLogin: 0,
			mockVaultURIPathRenewToken:   0,
		},
	}
	return requests, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.Method][r.URL.Path]++

		switch {
		case r.Method == "PUT" && r.URL.Path == mockVaultURIPathApproleLogin:
			mockVaultLogin(t)(w, r)
		case r.Method == "PUT" && r.URL.Path == mockVaultURIPathRenewToken:
			mockVaultTokenRenewal(t)(w, r)
		case r.Method == "GET" && r.URL.Path == mockVaultURIPathSTSCredentials:
			mockVaultSTSRequest(t)(w, r)
		default:
			t.Fatalf("Got unexpected Vault request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func mockVaultBrokenSTS(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mockVaultURIPathSTSCredentials {
			_, ts := mockVault(t)
			ts.Config.Handler.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
}

func mockVaultLogin(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)

		var requestData struct {
			RoleID string `json:"role_id"`
		}
		err := json.Unmarshal(body, &requestData)
		assert.NoError(t, err)
		assert.Equal(t, requestData.RoleID, testVaultRoleID)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"auth": map[string]interface{}{
				"client_token":   testVaultToken,
				"lease_duration": testVaultTokenLeaseDuration,
			},
		})
	}
}

func mockVaultTokenRenewal(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Vault-Token")
		assert.Equal(t, token, testVaultToken)

		body, _ := ioutil.ReadAll(r.Body)

		var requestData struct {
			Increment int `json:"increment"`
		}

		err := json.Unmarshal(body, &requestData)
		assert.NoError(t, err)
		assert.Equal(t, requestData.Increment, testVaultTokenLeaseDuration)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"auth": map[string]interface{}{
				"client_token":   testVaultToken,
				"lease_duration": testVaultTokenLeaseDuration,
			},
		})
	}
}

func mockVaultSTSRequest(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Vault-Token")
		assert.Equal(t, token, testVaultToken)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"access_key":     testAWSAccessKeyID,
				"secret_key":     testAWSSecretAccessKey,
				"security_token": testAWSSessionToken,
			},
			"lease_duration": testAWSCredentialsLeaseDuration,
		})
	}
}
