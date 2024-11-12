package joern

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/net/websocket"
	"net/http"
	"strings"
	"time"
)

type (
	Client struct {
		http                *http.Client
		ws                  *websocket.Conn
		baseURL, user, pass string
		bufferSize          int
		timeout             time.Duration
	}

	QueryRequest struct {
		Query string `json:"query"`
	}

	QueryResponse struct {
		UUID uuid.UUID `json:"uuid"`
	}

	Bool bool

	ResultResponse struct {
		Success bool   `json:"success"`
		Stdout  string `json:"stdout"`
		Stderr  string `json:"stderr"`
	}

	Option func(*Client)
)

func NewClient(options ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		bufferSize: defaultBufferSize,
		timeout:    defaultTimeout,
	}

	for _, option := range options {
		option(c)
	}

	c.http = &http.Client{
		Timeout: c.timeout,
	}

	c.http.Transport = roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		transport := http.DefaultTransport

		request.URL.Scheme = "http"
		request.URL.Host = c.baseURL

		if strings.TrimSpace(c.user) != "" && strings.TrimSpace(c.pass) != "" {
			request.SetBasicAuth(c.user, c.pass)
		}

		return transport.RoundTrip(request)
	})

	return c
}

func WithBaseURL(baseURL string) func(client *Client) {
	return func(client *Client) {
		client.baseURL = baseURL
	}
}

func WithBasicAuth(user, pass string) func(client *Client) {
	return func(client *Client) {
		client.user = user
		client.pass = pass
	}
}

func WithBufferSize(n int) func(client *Client) {
	return func(client *Client) {
		client.bufferSize = n
	}
}

func WithTimeout(timeout time.Duration) func(client *Client) {
	return func(client *Client) {
		client.timeout = timeout
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return r(request)
}

func (c *Client) Open(ctx context.Context) (err error) {
	config, err := websocket.NewConfig("ws://"+c.baseURL+"/connect", "ws://"+c.baseURL+"/connect")
	if err != nil {
		return err
	}
	c.ws, err = config.DialContext(ctx)
	return
}

func (c *Client) Close() (err error) {
	err = c.ws.Close()
	if err != nil {
		return
	}

	c.ws = nil

	return
}

func (c *Client) Send(ctx context.Context, query string) (response QueryResponse, err error) {
	pl := QueryRequest{Query: query}

	body, err := json.Marshal(pl)
	if err != nil {
		return
	}

	request, err := http.NewRequest("POST", "/query", bytes.NewBuffer(body))
	if err != nil {
		return
	}

	request = request.WithContext(ctx)

	res, err := c.http.Do(request)
	if err != nil {
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return
	}

	return
}

func (c *Client) Result(ctx context.Context, uuid uuid.UUID) (result ResultResponse, err error) {
	request, err := http.NewRequest("GET", "/result/"+uuid.String(), nil)
	if err != nil {
		return
	}
	request = request.WithContext(ctx)
	res, err := c.http.Do(request)
	if err != nil {
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return
	}

	return
}

func (c *Client) Receive(ctx context.Context, ch chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		buffer := make([]byte, c.bufferSize)

		read, err := c.ws.Read(buffer)
		if err != nil {
			return
		}
		if read > 0 {
			res := buffer[:read]
			ch <- string(res)
		}
	}
}

const (
	Connected         string = "connected"
	defaultTimeout           = 3600 * time.Second
	defaultBufferSize        = 36
	defaultBaseURL           = "localhost:8080"
)
