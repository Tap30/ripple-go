package adapters

// HTTPResponse represents the response from an HTTP request.
type HTTPResponse struct {
	OK     bool
	Status int
	Data   any
}

// HTTPAdapter is an interface for HTTP communication.
// Implement this interface to use custom HTTP clients.
type HTTPAdapter interface {
	// Send events to the specified endpoint.
	//
	// Parameters:
	//   - endpoint: The API endpoint URL
	//   - events: Array of events to send
	//   - headers: Optional custom headers to merge with defaults
	//
	// Returns HTTP response or error.
	Send(endpoint string, events []Event, headers map[string]string) (*HTTPResponse, error)
}
