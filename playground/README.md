# Ripple Go Playground

A testing environment for the Ripple Go SDK with a dummy HTTP server.

## Structure

- `cmd/server/main.go` - HTTP server that receives and logs events
- `cmd/client/main.go` - Interactive CLI client for manual testing

## Usage

### Start the Server

```bash
cd playground
go run cmd/server/main.go
```

Using Makefile:

```bash
make server
```

The server will start on `http://localhost:3000` and accept events at `/events`.

### Run the Client

For interactive testing with a CLI menu:

```bash
cd playground
go run cmd/client/main.go
```

Or using Makefile:

```bash
make client
```

The client provides a menu to:

**📊 Basic Event Tracking**
- Track Simple Event
- Track Event with Payload  
- Track Event with Metadata
- Track Event with Custom Metadata

**🏷️ Metadata Management**
- Set Shared Metadata
- Track with Shared Metadata
- View Current Context/Metadata

**📦 Batch and Flush**
- Track Multiple Events (Batch Test)
- Manual Flush

**⚠️ Error Handling**
- Test Retry Logic (Error Event)
- Test Invalid Endpoint

**🔄 Lifecycle Management**
- Dispose Client
- Exit

Example session:
```
🎯 Ripple Interactive Client
Connected to: http://localhost:3000/events

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Basic Event Tracking
1. Track Simple Event
2. Track Event with Payload
3. Track Event with Metadata
4. Track Event with Custom Metadata

🏷️  Metadata Management
5. Set Shared Metadata
6. Track with Shared Metadata
7. View Current Context/Metadata

📦 Batch and Flush
8. Track Multiple Events (Batch Test)
9. Manual Flush

⚠️  Error Handling
10. Test Retry Logic (Error Event)
11. Test Invalid Endpoint

🔄 Lifecycle Management
12. Dispose Client
13. Exit
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Choose an option: 1

📊 Track Simple Event
✅ Tracked: button_click

Choose an option: 5

🏷️  Set Shared Metadata
✅ Shared metadata set: key_1 = value_1
```

### Expected Output

**Server:**

```txt
🚀 Event tracking server running at http://localhost:3000
📍 Endpoint: http://localhost:3000/events
🔑 API Key: test-api-key
📊 Received events:
{
  "events": [
    {
      "name": "button_click",
      "payload": null,
      "issuedAt": 1734622890,
      "context": {},
      "metadata": {},
      "platform": { "type": "server" }
    }
  ]
}
```

**Client:**

```txt
📊 Track Simple Event
✅ Tracked: button_click

🔄 Flushing events...
✅ Events flushed
```

## E2E Testing

This playground is useful for:

- Manual testing of the SDK with comprehensive menu options
- Verifying event delivery and batching behavior
- Testing retry logic with simulated server errors
- Testing persistence (events saved to `ripple_events.json`)
- Testing invalid endpoints and error handling
- Debugging event payloads and metadata
- Testing shared metadata functionality

## Server Endpoints

### POST /events

Accepts events in the following format:

```json
{
  "events": [
    {
      "name": "event_name",
      "payload": {},
      "issuedAt": 1234567890,
      "context": {},
      "metadata": {},
      "platform": { "type": "server" }
    }
  ]
}
```

Returns:

```json
{
  "success": true,
  "received": 3
}
```

## Testing Scenarios

### 1. Normal Flow

```bash
# Terminal 1
go run cmd/server/main.go

# Terminal 2
go run cmd/client/main.go
```

### 2. Test Retry Logic

```bash
# Terminal 1
go run cmd/server/main.go

# Terminal 2
go run cmd/client/main.go
# Choose option 10 to test retry logic with error events
# Server will return 500 error and client will retry

# Or stop server (Ctrl+C) before flush
# Events should be persisted to ripple_events.json

# Restart server
go run cmd/server/main.go

# Run client again - persisted events should be sent
go run cmd/client/main.go
```

### 3. Test Batching

```bash
# Use option 8 in the client menu to track 10 events
# Observe auto-flush at batch size 5
```

### 4. Test Invalid Endpoint

```bash
# Use option 11 in the client menu
# Creates a client with invalid endpoint to test error handling
```
