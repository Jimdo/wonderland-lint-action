package vault

import (
	"net/http"
	"net/url"

	"github.com/jarcoal/httpmock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecretProvider", func() {

	const (
		mockResponse200 = `{"lease_id":"secret/foo/af5f7784-0b76-f1ee-2dd2-1b7d7e71a505","renewable":false,"lease_duration":2592000,"data":{"KEY1":"foo","KEY2":"bar","key3":"goo"},"auth":null}`
		mockResponse403 = `{"errors":["permission denied"]}`
		mockResponse404 = `{"errors":[]}`

		validToken = "f3b09679-3001-009d-2b80-9c306ab81aa6"
	)

	var (
		requestURL *url.URL
		vars       map[string]string
		err        error
	)

	BeforeEach(func() {
		httpmock.Activate()
		httpmock.RegisterResponder("GET", "https://127.0.0.1:8200/v1/secret/foo", func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Vault-Token") == validToken {
				return httpmock.NewStringResponse(200, mockResponse200), nil
			}
			return httpmock.NewStringResponse(403, mockResponse403), nil
		})
		httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Vault-Token") == validToken {
				return httpmock.NewStringResponse(404, mockResponse404), nil
			}
			return httpmock.NewStringResponse(403, mockResponse403), nil
		})
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	Describe("Fetching environment variables from Vault", func() {
		JustBeforeEach(func() {
			p := SecretProvider{}
			vars, err = p.GetValues(requestURL)
		})

		Context("With valid token and existing path", func() {
			BeforeEach(func() {
				requestURL, _ = url.Parse("vault+secret://token:f3b09679-3001-009d-2b80-9c306ab81aa6@127.0.0.1:8200/foo")
			})

			It("should not have errored", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have returned three keys", func() {
				Expect(len(vars)).To(Equal(3))
			})

			It("should have correct values", func() {
				Expect(vars["KEY1"]).To(Equal("foo"))
				Expect(vars["KEY2"]).To(Equal("bar"))
				Expect(vars["key3"]).To(Equal("goo"))
			})
		})

		Context("With valid token and not existing path", func() {
			BeforeEach(func() {
				requestURL, _ = url.Parse("vault+secret://token:f3b09679-3001-009d-2b80-9c306ab81aa6@127.0.0.1:8200/anyotherpath")
			})

			It("should have errored", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("With invalid token and existing path", func() {
			BeforeEach(func() {
				requestURL, _ = url.Parse("vault+secret://token:somethingdifferent@127.0.0.1:8200/foo")
			})

			It("should have errored", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Without any auth", func() {
			BeforeEach(func() {
				requestURL, _ = url.Parse("vault+secret://127.0.0.1:8200/foo")
			})

			It("should have errored", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("You need to provide a token"))
			})
		})

		Context("Without token", func() {
			BeforeEach(func() {
				requestURL, _ = url.Parse("vault+secret://token@127.0.0.1:8200/foo")
			})

			It("should have errored", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("You need to provide a token"))
			})
		})
	})

	Describe("Fetching environment variables from Vault with access token", func() {
		JustBeforeEach(func() {
			p := SecretProvider{
				VaultAccessToken: validToken,
			}
			vars, err = p.GetValues(requestURL)
		})

		Context("With global Vault-Access-Token", func() {
			BeforeEach(func() {
				requestURL, _ = url.Parse("vault+secret://127.0.0.1:8200/foo")
			})

			It("should not have errored", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have returned three keys", func() {
				Expect(len(vars)).To(Equal(3))
			})

			It("should have correct values", func() {
				Expect(vars["KEY1"]).To(Equal("foo"))
				Expect(vars["KEY2"]).To(Equal("bar"))
				Expect(vars["key3"]).To(Equal("goo"))
			})
		})
	})

})
