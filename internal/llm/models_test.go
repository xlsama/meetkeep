package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/xlsama/meetkeep/internal/config"
)

func TestListModels(t *testing.T) {
	var auth, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		w.Write([]byte(`{"object":"list","data":[{"id":"b-large"},{"id":""},{"id":"a-small","owned_by":"x"}]}`))
	}))
	defer server.Close()

	got, err := listModels(context.Background(), server.Client(), server.URL+"/v1/", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"b-large", "a-small"}; !slices.Equal(got, want) {
		t.Errorf("models = %v, want %v", got, want)
	}
	if path != "/v1/models" {
		t.Errorf("path = %q, want /v1/models", path)
	}
	if auth != "Bearer secret" {
		t.Errorf("Authorization = %q, want the key as a bearer token", auth)
	}

	if _, err := listModels(context.Background(), server.Client(), server.URL, ""); err != nil {
		t.Fatal(err)
	}
	if auth != "" {
		t.Errorf("Authorization = %q without a key, want none", auth)
	}
}

func TestListModelsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid key"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := listModels(context.Background(), server.Client(), server.URL, "wrong")
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid key") {
		t.Errorf("err = %v, want the status and what the service said", err)
	}
}

func TestModelsNotConfigured(t *testing.T) {
	for _, s := range []config.LLMSettings{
		{Provider: config.LLMAnthropic},
		{Provider: config.LLMDeepSeek},
		{Provider: config.LLMCustom, Service: "a", Services: []config.Service{{ID: "a", Key: "k"}, {ID: "b", BaseURL: "http://localhost:1/v1"}}},
	} {
		if _, err := Models(context.Background(), s); !errors.Is(err, ErrNotConfigured) {
			t.Errorf("%s: err = %v, want ErrNotConfigured", s.Provider, err)
		}
	}
}
