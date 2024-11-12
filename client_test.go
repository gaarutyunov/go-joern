package joern

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClient(t *testing.T) {
	client := NewClient(
		WithBaseURL(defaultBaseURL),
		WithBasicAuth("", ""),
		WithBufferSize(defaultBufferSize),
		WithTimeout(defaultTimeout),
	)

	ctx, cancel := context.WithCancel(context.Background())

	err := client.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	msgCh := make(chan string)
	go client.Receive(ctx, msgCh)
	defer func() {
		err := client.Close()
		if err != nil {
			t.Fatal(err)
		}
		close(msgCh)
	}()

	res, err := client.Send(ctx, "help")
	if err != nil {
		t.Fatal(err)
	}

	var i int

	for {
		select {
		case msg := <-msgCh:
			switch i {
			case 0:
				assert.Equal(t, Connected, msg)
			case 1:
				assert.Equal(t, res.UUID.String(), msg)
				result, err := client.Result(ctx, res.UUID)
				if err != nil {
					t.Fatal(err)
				}
				assert.True(t, result.Success)
				assert.Contains(t, result.Stdout, "help")
				assert.Empty(t, result.Stderr)
				cancel()
			}
			i++
		case <-ctx.Done():
			return
		}
	}
}
