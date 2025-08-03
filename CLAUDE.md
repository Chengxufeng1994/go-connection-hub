# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Run
- `go build -o ./tmp/main ./cmd` - Build the application to ./tmp/main
- `go run ./cmd` - Run the application directly
- `air` - Hot reload development server (requires air to be installed)

### Development with Air
The project uses Air for hot reloading during development. Configuration is in `.air.toml`:
- `air` - Start development server with auto-reload
- Build output goes to `./tmp/main`
- Excludes test files and certain directories from watch

## Architecture

This is a Go-based Server-Sent Events (SSE) notification service using Clean Architecture principles.

### Project Structure
- `cmd/` - Application entry point and routing configuration
- `internal/` - Private application code following Clean Architecture layers:
  - `domain/` - Business entities, events, services, and value objects
  - `application/` - Use cases, commands, queries, DTOs, and handlers
  - `infrastructure/` - External concerns (database, logging, event bus, HTTP server)
  - `interfaces/` - Controllers and adapters (REST API, SSE handlers)
  - `port/` - Interfaces for inbound/outbound communication
  - `adapter/` - Implementation of ports

### Key Components
- **HTTP Server**: Gin-based HTTP server running on port 8080 with graceful shutdown
- **SSE Endpoint**: `/sse` endpoint for Server-Sent Events connections
- **Logger**: Logrus-based structured logging with configurable levels and output
- **Event Architecture**: Placeholder for event bus and hub components

### Application Lifecycle
1. Main creates logger and router
2. HTTP server starts with graceful shutdown handling
3. Signal handling for SIGINT/SIGTERM with 5-second shutdown timeout
4. Uses errgroup for concurrent server management

### SSE Implementation
- SSE handlers in `internal/interfaces/sse/`
- Middleware sets proper SSE headers (Content-Type, Cache-Control, Connection)
- Handler structure is in place but ConnectSSE method is currently empty

### Dependencies
- **Gin**: HTTP web framework
- **Logrus**: Structured logging
- **golang.org/x/sync**: Error groups for concurrent operations
- **Lumberjack**: Log rotation

### Current State
The project appears to be a skeleton/template with the main infrastructure in place but core SSE functionality not yet implemented. The hub and eventbus packages are empty, and the SSE handler method is stubbed out.