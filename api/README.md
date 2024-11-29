## API package documentation

The api package implements the API layer of the application, providing a structured approach to manage HTTP servers, routes, handlers, and request/response mappings.

## Components

### Server
The Server struct manages the lifecycle of the HTTP server. It is responsible for:

- Listening for incoming requests.
- Serving responses via the [http.Server](https://pkg.go.dev/net/http#Server).
- Gracefully shutting down the server when required.

This component is highly configurable, allowing you to set:

- Server address and port.
- Read and write timeouts.

### Router

The default router is based on the lightweight [Chi router](https://go-chi.io/#/README), which can be replaced with any router implementing the [http.Handler](https://pkg.go.dev/net/http#Handler) interface.

Features:

- [Route Initialization](https://github.com/softika/gopherizer/blob/884d805e6adabedf965c2e7ee4569a11012a97ff/api/router.go#L19): Configures application routes and middleware.
- [Centralized Error Handling](https://github.com/softika/gopherizer/blob/884d805e6adabedf965c2e7ee4569a11012a97ff/api/router.go#L51): Provides a unified approach for managing errors across routes.

### Handler

The `Handler` struct serves as a generic interface for handling incoming HTTP requests. It decouples business logic from the HTTP layer by:

- Validating and passing the request's context and inputs to the service functions.
- Returning a response or error from the service functions.

How to Use:

- [Create a New Handler](https://github.com/softika/gopherizer/blob/884d805e6adabedf965c2e7ee4569a11012a97ff/api/bootstrap.go#L54)
- [Initialize Endpoints](https://github.com/softika/gopherizer/blob/884d805e6adabedf965c2e7ee4569a11012a97ff/api/routes.go#L47)

### Mappers

The Mappers struct simplifies the process of translating HTTP requests into service-layer inputs and mapping service outputs to HTTP responses.

Key Features:

- No need to create new handlers for each endpoint.
- Implement the following interfaces for custom mappings:
  - [RequestMapper](https://github.com/softika/gopherizer/blob/884d805e6adabedf965c2e7ee4569a11012a97ff/api/handler.go#L14)
  - [ResponseMapper](https://github.com/softika/gopherizer/blob/884d805e6adabedf965c2e7ee4569a11012a97ff/api/handler.go#L19)

**Examples**:</br>
The mappers package includes examples for handling various types of requests and responses.
