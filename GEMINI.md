# Gemini Code Assistant Context

## Project Overview

This project, "tok", is a Go library designed to simplify the creation of real-time Instant Messaging (IM) applications. It provides a flexible and modular framework for handling WebSocket and TCP connections, managing user presence, and routing messages.

The core of the library is the `Hub`, which acts as a central message dispatcher. It manages device connections, handles online/offline status, and uses a pluggable `Queue` interface for caching offline messages. The library is designed to be extensible, allowing developers to inject their own logic through various handlers and the main `Actor` interface.

## Key Technologies

*   **Language:** Go
*   **Primary Dependencies:**
    *   `github.com/gorilla/websocket`
    *   `github.com/coder/websocket`
    *   `golang.org/x/net/websocket`
*   **Testing:**
    *   `github.com/onsi/ginkgo/v2`
    *   `github.com/onsi/gomega`
*   **Mocking:**
    *   `go.uber.org/mock` (used with `go generate`)

## Building and Running

This project uses a `Makefile` to streamline common development tasks.

*   **To format code and tidy dependencies:**
    ```bash
    make fmt
    ```

*   **To generate mocks (required before testing/building):**
    ```bash
    make mock
    ```

*   **To run the test suite:**
    ```bash
    make test
    ```

*   **To build the project:**
    ```bash
    make build
    ```

## Development Conventions

*   **Architecture:** The library follows a modular design, separating network connection logic (`tcp_conn.go`, `ws_conn.go`) from the core message hub (`hub.go`). Business logic is implemented by the user by conforming to the `Actor` interface.
*   **Extensibility:** Functionality can be customized by implementing various handler interfaces, such as `BeforeReceiveHandler`, `BeforeSendHandler`, `CloseHandler`, and `PingGenerator`.
*   **Message Queuing:** An in-memory queue is provided (`memory_q.go`), but any queue that satisfies the `Queue` interface (`q.go`) can be integrated for offline message persistence.
*   **Testing:** Tests are written using the Ginkgo BDD framework. The `make test` command ensures that code is formatted and mocks are generated before running the tests.
*   **Single Sign-On (SSO):** The hub is configured by default to enforce SSO, meaning only one active connection is allowed per user ID.
