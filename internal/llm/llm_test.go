package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompleteQuickFallsBack(t *testing.T) {
	var asked []bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, quick := body["thinking"]
		asked = append(asked, quick)
		if quick {
			http.Error(w, `{"error":"Unrecognized request argument supplied: thinking"}`, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{}"}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "key", "model")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if answer, err := client.CompleteQuick(context.Background(), "system", "user"); err != nil || answer != "{}" {
			t.Fatalf("answered %q, %v", answer, err)
		}
	}
	if len(asked) != 3 || !asked[0] || asked[1] || asked[2] {
		t.Errorf("the requests asked not to think: %v, want only the first", asked)
	}
	if used := client.Used(); used.Requests != 2 || used.PromptTokens != 6 {
		t.Errorf("used %+v", used)
	}
}
