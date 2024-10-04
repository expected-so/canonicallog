# canonical log

A canonical logging solution for Go inspired from https://stripe.com/blog/canonical-log-lines

## Install

```bash
go get github.com/expected-so/canonicallog
```

## HTTP

```go
package main

import (
    "net/http"
    "log/slog"
    "github.com/expected-so/canonicallog"
)

func main()  {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      canonicallog.LogAttr(r.Context(), slog.String("user_id", "123"))
      w.WriteHeader(http.StatusNoContent)
    })
    http.ListenAndServe("0.0.0.0:3000", canonicallog.HttpHandler(handler))
}
```
