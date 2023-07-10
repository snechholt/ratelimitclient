package ratelimitclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitClientRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	var return429 int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { return429-- }()
		if return429 > 0 {
			w.WriteHeader(429)
		} else {
			w.WriteHeader(200)
		}
	}))
	client := NewClient(context.Background(), ts.Client(), 100, time.Second)

	assertResponse := func(testName string, wantStatusCode int) bool {
		req, err := http.NewRequest("GET", ts.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Test '%s': error: %v", testName, err)
		}
		if resp.StatusCode != wantStatusCode {
			t.Errorf("Test '%s': Wrong status code: want %v, got %v", testName, wantStatusCode, resp.Status)
		}
		return !t.Failed()
	}

	// When client.Retries = 0, do not retry requests and return original response when server returns 429
	client.Retries = 0
	return429 = 1 // Return 429 one time
	assertResponse("No retries", 429)

	// When client.Retries > N (N > 0), retry N times
	client.Retries = 2
	return429 = client.Retries
	assertResponse("Successful retries", 200)

	// When client.Retries > N (N > 0), retry N times, then return original response when request N + 1 also returns 429
	client.Retries = 2
	return429 = client.Retries + 1
	assertResponse("Unsuccessful retries", 429)
}

/*
type testClient struct {
	response *http.Response
	delay    time.Duration
	counter  int32
}

func (tc *testClient) Do(*http.Request) (*http.Response, error) {
	atomic.AddInt32(&tc.counter, 1)
	time.Sleep(tc.delay)
	return tc.response, nil
}

func TestRunRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// The test runs for a few seconds executing many requests and then checks
	// that overall number of requests is reasonable.
	const (
		limit = 100
		unit  = time.Second
	)

	tc := &testClient{
		response: &http.Response{
			StatusCode: 200,
		},
	}
	client := NewClient(context.Background(), tc, limit, unit)

	f := func() {
		resp, err := client.Do(nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp != tc.response {
			t.Fatal("Wrong result returned")
		}
	}

	start := time.Now()
	end := start.Add(5 * time.Second)
	for time.Now().Before(end) {
		go f()

		// This will still offer ~500 requests per second, but won't consume
		// outrageous amount of CPU.
		time.Sleep(2 * time.Millisecond)
	}
	elapsed := time.Since(start)
	ideal := 1 + (limit * float64(elapsed) / float64(time.Second))

	// We should never get more requests than allowed.
	if want := int32(ideal + 1); tc.counter > want {
		t.Errorf("tc.counter = %d, want %d (ideal %f)", tc.counter, want, ideal)
	}
	// We should get very close to the number of requests allowed.
	if want := int32(0.999 * ideal); tc.counter < want {
		t.Errorf("tc.counter = %d, want %d (ideal %f)", tc.counter, want, ideal)
	}
}

func TestRunRequest2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// The test runs for a few seconds executing many requests and then checks
	// that overall number of requests is reasonable.
	const (
		limit = 100
		unit  = time.Second

		delay = 4 * time.Second
	)

	tc := &testClient{
		delay: delay,
		response: &http.Response{
			StatusCode: 200,
		},
	}
	client := NewClient(context.Background(), tc, limit, unit)

	f := func() {
		resp, err := client.Do(nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp != tc.response {
			t.Fatal("Wrong result returned")
		}
	}

	start := time.Now()
	end := start.Add(5 * time.Second)
	for time.Now().Before(end) {
		go f()

		// This will still offer ~500 requests per second, but won't consume
		// outrageous amount of CPU.
		time.Sleep(2 * time.Millisecond)
	}

	if want := 200; tc.counter != int32(want) {
		t.Errorf("tc.counter = %d, want %d", tc.counter, want)
	}
}
*/
