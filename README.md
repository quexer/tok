tok
===

"talk", a library to simplify creating IM application

Installation
------
    go get github.com/quexer/tok


Features
--------

- Supports both TCP and WebSocket servers for flexible IM application deployment.
- Modular design: pluggable network adapters(ConAdapter interface), making it easy to extend or customize.
- Simple API for creating hubs, managing connections, and handling messages.
- Built-in memory queue for offline message caching, with pluggable queue interface.
- Supports single sign-on (SSO) to ensure only one active connection per user.
- Configurable timeouts for authentication, server ping, and message reading/writing.
- Easy integration with custom authentication logic.
- Cluster support available via [quexer/cluster](https://github.com/quexer/cluster).
- Graceful connection lifecycle management with context-based cancellation.
- Built-in OpenTelemetry observability: metrics and traces for connections, messages, and queue operations.

Observability
-------------

tok has built-in [OpenTelemetry](https://opentelemetry.io/) support for metrics and tracing. By default it uses the global noop providers; supply your own to enable observability:

```go
cfg, _ := tok.NewHubConfig(actor,
    tok.WithMeterProvider(myMeterProvider),
    tok.WithTracerProvider(myTracerProvider),
)
```

### Metrics

| Name | Type | Description |
|------|------|-------------|
| `tok.connections.online` | UpDownCounter | Current number of online connections |
| `tok.messages.up` | Counter | Total messages received from clients |
| `tok.messages.down` | Counter | Total messages sent to clients |
| `tok.queue.enqueue` | Counter | Total messages enqueued to offline queue |
| `tok.queue.dequeue` | Counter | Total messages dequeued from offline queue |
| `tok.send.duration` | Histogram (seconds) | Duration of Hub.Send operations |

### Traces

| Span Name | Location |
|-----------|----------|
| `tok.Auth` | TCP/WebSocket authentication phase |
| `tok.Send` | Hub.Send message delivery |
| `tok.Receive` | Upstream message handling (Actor.OnReceive) |
| `tok.Cache` | Offline message enqueue |

WebSocket Engine Support
-----------------------

tok supports multiple WebSocket engines for flexible integration:

- `golang.org/x/net/websocket` (default)
- `github.com/gorilla/websocket`
- `github.com/coder/websocket` ( former `nhooyr.io/websocket`)

You can select the engine via configuration options. Future engines can be added easily.

Architecture
------------

```
                          Tok Framework Architecture
                         ===============================

┌─────────────────────────────────────────────────────────────────────────┐
│                             Client Layer                                │
└─────────────┬─────────────────┬─────────────────┬───────────────────────┘
              │                 │                 │
         TCP Client       WebSocket Client   Custom Client
              │                 │                 │
┌─────────────┴─────────────────┴─────────────────┴───────────────────────┐
│                          Network Layer                                  │
│                                                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐          │
│  │   tcpAdapter    │  │   wsAdapter     │  │  CustomAdapter  │          │
│  │                 │  │  (x/gorilla/    │  │                 │          │
│  │   - Read()      │  │   coder)        │  │   - Read()      │          │
│  │   - Write()     │  │   - Read()      │  │   - Write()     │          │
│  │   - Close()     │  │   - Write()     │  │   - Close()     │          │
│  │   - ShareConn() │  │   - Close()     │  │   - ShareConn() │          │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘          │
│                                                                         │
└─────────────┬─────────────────┬─────────────────┬───────────────────────┘
              │                 │                 │
              └─────────────────┼─────────────────┘
                                │
                        ConAdapter Interface
                                │
┌───────────────────────────────┴─────────────────────────────────────────┐
│                          Connection Layer                               │
│                                                                         │
│    ┌─────────────────────────────────────────────────────────────────┐  │
│    │                        connection                               │  │
│    │                                                                 │  │
│    │    - adapter: ConAdapter                                        │  │
│    │    - dv: *Device (User + Device metadata)                       │  │
│    │    - hub: *Hub                                                  │  │
│    │    - readLoop()      ← blocking read                            │  │
│    │    - Write()         ← thread-safe write                        │  │
│    │    - triggerOffline() ← atomic state change                     │  │
│    │                                                                 │  │
│    └─────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
┌───────────────────────────────┴─────────────────────────────────────────┐
│                        Hub (Message Dispatcher)                         │
│                                                                         │
│  Channel-based Architecture:                                            │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  chUp          ← upstream messages                                │  │
│  │  chDown        ← downstream messages                              │  │
│  │  chConState    ← connection state changes                         │  │
│  │  chReadSignal  ← trigger offline message delivery                 │  │
│  │  chKick        ← kick user connections                            │  │
│  │  chCheck       ← online status queries                            │  │
│  │  chQueryOnline ← get online user list                             │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  Connection Management:                                                 │
│  cons map[interface{}][]*connection  ← user_id -> connections           │
│                                                                         │
│  Features:                                                              │
│  • SSO (Single Sign-On) support                                         │
│  • Server-side ping with configurable intervals                         │
│  • Read/Write timeout management                                        │
│  • Message caching via Queue interface                                  │
│                                                                         │
└─────────────┬─────────────────────────────────────┬─────────────────────┘
              │                                     │
┌─────────────┴─────────────────────────────────────┴─────────────────────┐
│                       Business Logic Layer                              │
│                                                                         │
│ ┌─────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐   │
│ │     Actor       │  │ Optional Handlers   │  │      Queue          │   │
│ │                 │  │                     │  │    Interface        │   │
│ │  OnReceive()    │  │  BeforeReceive      │  │                     │   │
│ │  (Required)     │  │  BeforeSend         │  │  Enq() / Deq()      │   │
│ │                 │  │  AfterSend          │  │                     │   │
│ │                 │  │  CloseHandler       │  │  MemoryQueue        │   │
│ │                 │  │  PingGenerator      │  │  (built-in)         │   │
│ │                 │  │  ByeGenerator       │  │                     │   │
│ └─────────────────┘  └─────────────────────┘  └─────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                       Observability (OpenTelemetry)                     │
│                                                                         │
│  Metrics: tok.connections.online, tok.messages.up/down,                 │
│           tok.queue.enqueue/dequeue, tok.send.duration                  │
│  Traces:  tok.Auth, tok.Send, tok.Receive, tok.Cache                   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

                               Message Flow
                              ===============

    Upstream (Client → Server):
    Client → Adapter → Connection → Hub → Actor.OnReceive()

    Downstream (Server → Client):
    Hub.Send() → Connection → Adapter → Client

    Connection Lifecycle:
    Connect → Auth → RegisterConnection → readLoop (blocking)
            → goOnline → Ping Loop (if enabled)
            → goOffline → Close → Cleanup

```

Structure
---------

- `tok.go`         : Entry and core types for the library.
- `hub.go`         : Hub logic for managing connections and message dispatch.
- `hub_config.go`  : Hub configuration and options.
- `tcp_conn.go`    : TCP server and adapter implementation.
- `ws_conn.go`     : WebSocket server implementation supporting multiple engines.
- `ws_gorilla.go`  : `github.com/gorilla/websocket` adapter.
- `ws_x.go`        : `golang.org/x/net/websocket` adapter.
- `ws_coder.go`    : `github.com/coder/websocket` adapter.
- `ws_option.go`   : WebSocket engine selection and options.
- `memory_q.go`    : Built-in in-memory message queue for offline messages.
- `otel.go`        : OpenTelemetry instruments (metrics and traces).
- `device.go`      : Device abstraction for user device.
- `example/`       : Example server and client implementations. [See examples](./example/)
