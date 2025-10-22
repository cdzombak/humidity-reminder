package mailgun_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"humidity-reminder/internal/mailgun"
)

func TestSend(t *testing.T) {
	var received url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/testdomain/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("missing basic auth")
		}
		if username != "api" || password != "key-test" {
			t.Fatalf("unexpected credentials: %s %s", username, password)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		received = r.PostForm

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Queued"}`))
	}))
	defer server.Close()

	client := mailgun.NewClient("testdomain", "key-test", server.Client(), mailgun.WithBaseURL(server.URL))
	err := client.Send(context.Background(), "from@example.com", "to@example.com", "Humidity Update", "Body text")
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if received == nil {
		t.Fatal("no form data received")
	}

	for key, expected := range map[string]string{
		"from":    "from@example.com",
		"to":      "to@example.com",
		"subject": "Humidity Update",
		"text":    "Body text",
	} {
		if got := strings.Join(received[key], ","); got != expected {
			t.Fatalf("unexpected value for %s: %q", key, got)
		}
	}
}
