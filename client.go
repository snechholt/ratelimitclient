// Package ratelimitclient implements an HTTP client that applies
// rate limiting to requests.

package ratelimitclient

import (
	"context"
	"golang.org/x/time/rate"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	client HttpClient
	limit  int
	unit   time.Duration

	Retries int

	ctx         context.Context
	rateLimiter *rate.Limiter
	rateChan    chan struct{}
}

// NewClient returns a Client that rate limits requests through the provided HTTPClient.
func NewClient(ctx context.Context, client HttpClient, limit int, unit time.Duration) *Client {
	return &Client{
		client:      client,
		limit:       limit,
		unit:        unit,
		ctx:         ctx,
		Retries:     5,
		rateLimiter: rate.NewLimiter(rate.Every(unit/time.Duration(limit)), 1),
		rateChan:    make(chan struct{}, limit),
	}
}

// Do sends an HTTP request and returns an HTTP response using the underlying Client implementation
// and makes sure requests are performed within its specified rate limit.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.rateChan <- struct{}{}
	var noChanReceive bool
	defer func() {
		if !noChanReceive {
			<-c.rateChan
		}
	}()
	if err := c.rateLimiter.Wait(c.ctx); err != nil {
		return nil, err
	}
	for i := 0; true; i++ {
		resp, err := c.client.Do(req)
		if err != nil {
			return resp, err
		}

		if resp.StatusCode != 429 {
			noChanReceive = true
			body := resp.Body
			resp.Body = &readCloser{
				r: body,
				closeFn: func() error {
					defer func() { <-c.rateChan }()
					return body.Close()
				},
			}
			return resp, err
		}

		if i >= c.Retries {
			return resp, err
		}
		delay := time.Duration(rand.Float64() / 2 * float64(c.unit))
		time.Sleep(c.unit + delay)
	}
	panic("We should never get here")
}

type readCloser struct {
	r       io.Reader
	closeFn func() error
}

// Read reads from the underlying reader into p.
func (r *readCloser) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

// Close calls the close function provided when creating the reader.
func (r *readCloser) Close() error {
	if r.closeFn != nil {
		return r.closeFn()
	}
	return nil
}
