package ripple

import "testing"

func TestHTTPError_Error(t *testing.T) {
	err := &HTTPError{Status: 500}
	if err.Error() != "HTTP request failed" {
		t.Fatal("expected error message")
	}
}
