package adapters

import "context"

// HTTPResponse represents the response from an HTTP request.
type HTTPResponse struct {
	Status int
	Data   any
}

// HTTPAdapter is an interface for HTTP communication.
// Implement this interface to use custom HTTP clients.
type HTTPAdapter interface {
	// Send events to the specified endpoint without context.
	//
	// Parameters:
	//   - endpoint: The API endpoint URL
	//   - events: Array of events to send
	//   - headers: HTTP headers including API key
	//
	// Returns HTTP response or error.
	Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)

	// SendWithContext sends events to the specified endpoint with context support.
	//
	// Parameters:
	//   - ctx: Context for timeout/cancellation
	//   - endpoint: The API endpoint URL
	//   - events: Array of events to send
	//   - headers: HTTP headers including API key
	//
	// Returns HTTP response or error.
	SendWithContext(ctx context.Context, endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
}
