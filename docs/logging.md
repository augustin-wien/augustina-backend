# Request-scoped logging (request_id)

This project uses zap for structured logging. The `RequestLogger` middleware injects a request-scoped logger into the request context that includes the `request_id`.

Helpers:

- `utils.WithLogger(ctx, logger)` — store a logger in a context
- `utils.LoggerFromContext(ctx) *zap.SugaredLogger` — retrieve a logger from context; falls back to `utils.GetLogger()` if none is present

Middleware:

- `middlewares.RequestLogger` — the middleware attaches a logger with `request_id` to the request context. When this middleware runs, handlers can use the logger from the context to include `request_id` in all logs.

Usage example in an HTTP handler:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
    logger := utils.LoggerFromContext(r.Context())
    logger.Infow("handling request", "path", r.URL.Path)

    // do work
    if err := doSomething(); err != nil {
        logger.Errorw("error doing something", "error", err)
        utils.ErrorJSON(w, err, http.StatusInternalServerError)
        return
    }

    logger.Infow("done")
}
```

Notes & migration guidance:

- Prefer `utils.LoggerFromContext(r.Context())` in handlers and request-scoped code.
- Avoid package-level `var log = utils.GetLogger()` in code that handles requests — replace usage with the context logger so logs include `request_id`.
- Library or background code (outside HTTP request handling) may continue to use `utils.GetLogger()`.

If you want, I can help update more handlers to use the context logger (I can do this in small batches).