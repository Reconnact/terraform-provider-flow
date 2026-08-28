package flow

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/flowswiss/goclient"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Every mutating API call is retried with a bounded backoff, regardless of the
// error message: the Flow API reports transient conditions (parallel creates,
// servers still booting, ports still held) with the same 400 it uses for
// genuine mistakes. Not retried: 401/403, 404 (success on a delete), context
// cancellation, and transport failures on a create (the request may have gone
// through). The budget is short so that real mistakes still surface quickly;
// it is configurable via the provider's `retry_timeout`.

const (
	defaultRetryTimeout      = 90 * time.Second
	defaultRetryInitialDelay = 1 * time.Second
	defaultRetryMaxDelay     = 15 * time.Second
)

type retryPolicy struct {
	Timeout      time.Duration
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// Shared by all resources; Configure sets Timeout from the provider config.
var defaultRetryPolicy = retryPolicy{
	Timeout:      defaultRetryTimeout,
	InitialDelay: defaultRetryInitialDelay,
	MaxDelay:     defaultRetryMaxDelay,
}

type retryOperation int

const (
	opMutate retryOperation = iota
	opCreate
)

// retry runs fn until it succeeds or the budget is exhausted and returns the
// last error unchanged. A 404 is treated as success.
func retry(ctx context.Context, what string, fn func() error) error {
	return retryWith(ctx, defaultRetryPolicy, opMutate, what, fn)
}

// retryCreate is retry for calls that create objects: a transport failure is
// not retried because a duplicate could be created.
func retryCreate(ctx context.Context, what string, fn func() error) error {
	return retryWith(ctx, defaultRetryPolicy, opCreate, what, fn)
}

func retryWith(ctx context.Context, policy retryPolicy, op retryOperation, what string, fn func() error) error {
	start := time.Now()
	delay := policy.InitialDelay

	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		switch classifyRetry(err, op) {
		case retryTreatAsSuccess:
			tflog.Debug(ctx, "object already gone, treating as success", map[string]interface{}{
				"operation": what,
				"error":     err.Error(),
			})
			return nil
		case retryStop:
			return err
		}

		elapsed := time.Since(start)
		if elapsed+delay > policy.Timeout {
			tflog.Debug(ctx, "retry budget exhausted", map[string]interface{}{
				"operation": what,
				"attempts":  attempt,
				"elapsed":   elapsed.String(),
				"error":     err.Error(),
			})
			return err
		}

		tflog.Debug(ctx, "retrying after error", map[string]interface{}{
			"operation": what,
			"attempt":   attempt,
			"wait":      delay.String(),
			"status":    statusCode(err),
			"error":     err.Error(),
		})

		select {
		case <-time.After(jitter(delay)):
		case <-ctx.Done():
			return err
		}

		delay *= 2
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}
}

type retryDecision int

const (
	retryStop retryDecision = iota
	retryAgain
	retryTreatAsSuccess
)

func classifyRetry(err error, op retryOperation) retryDecision {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return retryStop
	}

	var apiErr goclient.APIError
	if errors.As(err, &apiErr) && apiErr.Response() != nil {
		switch apiErr.Response().StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return retryStop
		case http.StatusNotFound:
			if op == opMutate {
				return retryTreatAsSuccess
			}
			return retryStop
		default:
			return retryAgain
		}
	}

	if op == opCreate && isTransportError(err) {
		return retryStop
	}

	return retryAgain
}

// goclient wraps failures without a response as "do request: ..." and
// unparseable error bodies as "parse response body: ..."; only the latter
// means the API actually answered.
func isTransportError(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if strings.HasPrefix(e.Error(), "parse response body") {
			return false
		}
	}
	return true
}

func statusCode(err error) int {
	var apiErr goclient.APIError
	if errors.As(err, &apiErr) && apiErr.Response() != nil {
		return apiErr.Response().StatusCode
	}
	return 0
}

func jitter(d time.Duration) time.Duration {
	return d + time.Duration(rand.Int63n(int64(d)/4+1))
}

func parseRetryTimeout(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", value, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("retry timeout must not be negative, got %s", d)
	}
	return d, nil
}
