# Ripple Go Playground

A testing environment for the Ripple Go SDK with a dummy HTTP server.

## Structure

- `server.go` - HTTP server that receives and logs events
- `client.go` - Interactive CLI client for manual testing

## Usage

### Start the Server

```bash
cd playground
go run server.go
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
go run client.go
```

Or using Makefile:

```bash
make client
```

The client provides a menu to:
- **Set Context** - Automatically adds `key_i: value_i` (incremented)
- **View Context** - Display current context
- **Track Event** - Automatically creates `event_i` with sample payload
- **Flush Events** - Manually trigger event flush
- **Exit** - Gracefully shutdown

Example session:
```
🎯 Ripple Interactive Client
Connected to: http://localhost:3000/events

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. Set Context
2. View Context
3. Track Event
4. Flush Events
5. Exit
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Choose an option: 1

📝 Set Context
✅ Context set: key_1 = value_1

Choose an option: 3

📊 Track Event
✅ Event 'event_1' tracked with sample payload
```

### Expected Output

**Server:**

```txt
🚀 Event tracking server running at http://localhost:3000
📍 Endpoint: http://localhost:3000/events
🔑 API Key: Bearer test-api-key
📊 Received events:
{
  "events": [
    {
      "name": "page_view",
      "payload": { "page": "/home" },
      "issuedAt": 1234567890,
      "context": { "userId": "user-123" },
      "platform": { "type": "server" }
    }
  ]
}
```

**Client:**

```txt
📤 Tracking events...
✅ Events tracked. Waiting for flush...
🔄 Manual flush...
✨ Done!
```

## E2E Testing

This playground is useful for:

- Manual testing of the SDK
- Verifying event delivery
- Testing retry logic (stop/start server)
- Testing persistence (kill client before flush)
- Debugging event payloads

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
go run server.go

# Terminal 2
go run client.go
```

### 2. Test Retry Logic

```bash
# Terminal 1
go run server.go

# Terminal 2
go run client.go

# Stop server (Ctrl+C) before flush
# Events should be persisted to ripple_events.json

# Restart server
go run server.go

# Run client again - persisted events should be sent
go run client.go
```

### 3. Test Batching

Modify `client.go` to track more events and observe batching behavior.

### 4. Test Custom Adapters

Create custom HTTP or storage adapters and test them here.
