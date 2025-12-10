package adapters

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNetHTTPAdapter_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type: application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("expected Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	adapter := NewNetHTTPAdapter()
	events := []Event{{Name: "test"}}
	headers := map[string]string{"Authorization": "Bearer test-key"}

	resp, err := adapter.Send(server.URL, events, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.OK || resp.Status != 200 {
		t.Fatal("expected successful response")
	}
}

func TestNetHTTPAdapter_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewNetHTTPAdapter()
	events := []Event{{Name: "test"}}

	resp, err := adapter.Send(server.URL, events, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OK {
		t.Fatal("expected response to not be OK")
	}
	if resp.Status != 500 {
		t.Fatalf("expected status 500, got %d", resp.Status)
	}
}

func TestNetHTTPAdapter_SendInvalidURL(t *testing.T) {
	adapter := NewNetHTTPAdapter()
	events := []Event{{Name: "test"}}

	_, err := adapter.Send("http://invalid-url-that-does-not-exist-12345.com", events, nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNetHTTPAdapter_SendMarshalError(t *testing.T) {
	adapter := NewNetHTTPAdapter()
	events := []Event{{
		Name:    "test",
		Payload: map[string]interface{}{"invalid": make(chan int)},
	}}

	_, err := adapter.Send("http://test.com", events, nil)
	if err == nil {
		t.Fatal("expected error for unmarshalable data")
	}
}

func TestNetHTTPAdapter_SendInvalidMethod(t *testing.T) {
	adapter := NewNetHTTPAdapter()
	events := []Event{{Name: "test"}}

	_, err := adapter.Send("ht!tp://invalid", events, nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
