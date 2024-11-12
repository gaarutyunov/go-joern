# go-joern

Go client for Joern.

## Usage

1. Run Joern as a [Server](https://docs.joern.io/server/):

```bash
joern --server --server-host localhost \
               --server-port 8080 \
               --server-auth-username user \
               --server-auth-password pass
```

2. Connect using the client

```go
package main

import (
	"context"
	"github.com/gaarutyunov/go-joern"
	"github.com/google/uuid"
	"time"
)

func main() {
	client := joern.NewClient(
		joern.WithBaseURL("localhost:8080"),
		joern.WithBasicAuth("user", "pass"),
		joern.WithBufferSize(36),
		joern.WithTimeout(3600*time.Second),
	)
	ctx := context.Background()
	msg := make(chan string)

	err := client.Open(ctx)
	if err != nil {
		panic(err)
	}

	defer func() {
		client.Close()
		close(msg)
	}()

	go func() {
		for m := range msg {
			switch m {
			case joern.Connected:
			default:
				id, err := uuid.FromBytes([]byte(m))
				if err != nil {
					panic(err)
                }
				
				result, err := client.Result(ctx, id)
				if err != nil {
					panic(err)
				}
				
				println(result.Stdout)
			}
		}
	}()

	_, err = client.Send(ctx, "help")
	if err != nil {
		panic(err) 
	}

	client.Receive(ctx, msg)
}
```
