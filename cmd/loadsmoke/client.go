package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// client talks to the API as one athlete and records every request.
type client struct {
	base  string
	http  *http.Client
	rec   *recorder
	token string
}

// call is one request: what to send, what counts as success, and how it is
// reported.
type call struct {
	method, path string
	route        string // the route template, so ids do not split the stats
	class        class
	body         any
	headers      map[string]string
	want         []int // accepted statuses
	out          any   // decoded from a 2xx body when set
}

// do sends a request. A status outside want is an error, recorded and
// returned. Rate-limited auth calls wait out Retry-After and are not counted.
func (c *client) do(ctx context.Context, k call) (int, error) {
	var raw []byte
	if k.body != nil {
		var err error
		if raw, err = json.Marshal(k.body); err != nil {
			return 0, fmt.Errorf("%s: encoding body: %w", k.route, err)
		}
	}
	for {
		req, err := http.NewRequestWithContext(ctx, k.method, c.base+k.path, bytes.NewReader(raw))
		if err != nil {
			return 0, fmt.Errorf("%s: %w", k.route, err)
		}
		if raw != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		for h, v := range k.headers {
			req.Header.Set(h, v)
		}

		start := time.Now()
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return 0, fmt.Errorf("%s: %w", k.route, ctx.Err())
			}
			c.rec.add(sample{route: k.route, class: k.class, latency: time.Since(start)})
			c.rec.fail(fmt.Sprintf("%s: %v", k.route, err))
			return 0, fmt.Errorf("%s: %w", k.route, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		latency := time.Since(start)
		if readErr != nil && ctx.Err() != nil {
			// The run ended mid-response; that is not the API's failure.
			return 0, fmt.Errorf("%s: %w", k.route, ctx.Err())
		}

		if resp.StatusCode == http.StatusTooManyRequests && k.class == classAuth {
			wait, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			select {
			case <-time.After(time.Duration(max(wait, 1)) * time.Second):
				continue
			case <-ctx.Done():
				return 0, fmt.Errorf("%s: %w", k.route, ctx.Err())
			}
		}

		ok := readErr == nil && accepted(resp.StatusCode, k.want)
		if ok && k.out != nil && resp.StatusCode/100 == 2 {
			if err := json.Unmarshal(body, k.out); err != nil {
				ok = false
				readErr = fmt.Errorf("decoding: %w", err)
			}
		}
		c.rec.add(sample{route: k.route, class: k.class, latency: latency, ok: ok})
		if !ok {
			msg := fmt.Sprintf("%s: status %d: %s", k.route, resp.StatusCode, truncate(body, 300))
			if readErr != nil {
				msg = fmt.Sprintf("%s: %v", k.route, readErr)
			}
			c.rec.fail(msg)
			return resp.StatusCode, fmt.Errorf("%s", msg)
		}
		return resp.StatusCode, nil
	}
}

func decode(resp *http.Response, v any) error {
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decoding: %w", err)
	}
	return nil
}

func accepted(status int, want []int) bool {
	for _, w := range want {
		if status == w {
			return true
		}
	}
	return false
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}
