package testutils

import (
	"net/http"
	"net/http/httptest"

	"claimctl-cli/pkg/api"
)

// SetupTestClient creates a test client with a test server
func SetupTestClient(handler http.Handler) (*api.Client, *httptest.Server) {
	server := httptest.NewServer(handler)

	client, err := api.NewClient(server.URL, "test-token", false)
	if err != nil {
		panic("Failed to create test client: " + err.Error())
	}

	// Use the server's client which will route to our handler
	client.HttpClient = server.Client()

	return client, server
}

// CreateTestClientWithResponse creates a test client that returns a specific response
func CreateTestClientWithResponse(response string, statusCode int) (*api.Client, *httptest.Server) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	})

	return SetupTestClient(handler)
}

// CreateTestClient creates a basic test client with default responses
func CreateTestClient() (*api.Client, *httptest.Server) {
	return CreateTestClientWithResponse(`{}`, http.StatusOK)
}
