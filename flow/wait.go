package flow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// waiting for real state: the api marks an order as processed while the
// resource is still coming up (a server keeps booting for seconds to minutes),
// so callers poll the actual status with a deadline

// default deadline per kind of wait, used when the resource carries no
// `timeouts {}` block — a configured one replaces them, see withTimeout
const (
	defaultWaitInterval = 3 * time.Second
	serverBootTimeout   = 10 * time.Minute
	clusterWaitTimeout  = 20 * time.Minute
	volumeSettleTimeout = 5 * time.Minute
	// a load balancer create stays working for about a minute, pool and member
	// changes for seconds
	loadBalancerTimeout = 10 * time.Minute
	// snapshot create and volume restore copy the data and scale with its size
	snapshotTimeout = 30 * time.Minute
	// an order is processed within seconds — a stuck order worker would
	// otherwise keep the apply hanging until ctrl-c
	orderTimeout = 10 * time.Minute
	// the longest synchronous call is the volume detach, which the backend
	// holds for up to 30 seconds
	responseHeaderTimeout = 2 * time.Minute
	// a delete answers 204 while the teardown still runs — the next delete in
	// the dependency chain fails until the resource is really gone
	goneTimeout = 10 * time.Minute
)

type timeoutGetter func(context.Context, time.Duration) (time.Duration, diag.Diagnostics)

// withTimeout puts the resource's `timeouts {}` value for one operation on ctx
// as a deadline. The configured  value is the budget for the whole operation.
func withTimeout(ctx context.Context, get timeoutGetter, diagnostics *diag.Diagnostics) (context.Context, context.CancelFunc) {
	timeout, diags := get(ctx, 0)
	diagnostics.Append(diags...)

	if timeout <= 0 {
		return ctx, func() {}
	}

	tflog.Debug(ctx, "applying configured timeout", map[string]interface{}{
		"timeout": timeout.String(),
	})
	return context.WithTimeout(ctx, timeout)
}

// the generated docs give a `timeouts {}` value nothing but the duration format —
// say what the operation waits for and what bounds it when the value is unset
func timeoutDescription(what string) string {
	return what + `; a string that can be ` +
		`[parsed as a duration](https://pkg.go.dev/time#ParseDuration), such as "30s" or "2h45m"`
}

// remaining is how long a single wait may run: what is left of the operation's
// configured budget, or the wait's own default when none was configured
func remaining(ctx context.Context, fallback time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline)
	}
	return fallback
}

// the sdk polls an order until it succeeds, fails or the context ends — this
// bounds it and names the order in the error
func waitForOrder(ctx context.Context, service common.OrderService, ordering common.Ordering) (common.Order, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, remaining(ctx, orderTimeout))
	defer cancel()

	order, err := service.WaitUntilProcessed(ctx, ordering)
	if err == nil {
		return order, nil
	}

	id, _ := ordering.ExtractIdentifier()
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return order, fmt.Errorf("timeout after %s waiting for order %d to be processed", time.Since(start).Round(time.Second), id)
	case errors.Is(err, common.ErrOrderFailed):
		if order.Status.Name != "" {
			return order, fmt.Errorf("order %d failed (%s)", id, order.Status.Name)
		}
		return order, fmt.Errorf("order %d failed", id)
	}
	return order, err
}

type terminalError struct{ err error }

func (t terminalError) Error() string { return t.err.Error() }
func (t terminalError) Unwrap() error { return t.err }

func stopWaiting(err error) error {
	return terminalError{err: err}
}

// waitFor polls check until it reports done, the deadline passes or the
// context is cancelled — an error from check does not abort the wait, it only
// surfaces in the timeout error if it never went away, unless the check marks
// it terminal with stopWaiting
func waitFor(ctx context.Context, timeout, interval time.Duration, name string, check func(ctx context.Context) (bool, error)) error {
	start := time.Now()
	deadline := start.Add(remaining(ctx, timeout))
	var lastErr error

	for attempt := 1; ; attempt++ {
		done, err := check(ctx)
		if err == nil && done {
			return nil
		}
		if err != nil {
			var terminal terminalError
			if errors.As(err, &terminal) {
				return fmt.Errorf("waiting for %s: %w", name, terminal.err)
			}
			lastErr = err
		}

		if time.Now().Add(interval).After(deadline) {
			waited := time.Since(start).Round(time.Second)
			if lastErr != nil {
				return fmt.Errorf("timeout after %s waiting for %s (last error: %w)", waited, name, lastErr)
			}
			return fmt.Errorf("timeout after %s waiting for %s", waited, name)
		}

		tflog.Debug(ctx, "waiting", map[string]interface{}{
			"for":     name,
			"attempt": attempt,
			"error":   errString(err),
		})

		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return fmt.Errorf("cancelled while waiting for %s: %w", name, ctx.Err())
		}
	}
}

// waitForGone polls get until the api answers 404. A delete returns on the
// api's 204 with the teardown still running, and terraform starts the next
// delete as soon as this one returns: destroying a network right after the
// load balancer in front of it fails while Octavia is still tearing down.
func waitForGone(ctx context.Context, timeout time.Duration, name string, get func(ctx context.Context) error) error {
	return waitFor(ctx, timeout, defaultWaitInterval, name+" to be gone", func(ctx context.Context) (bool, error) {
		err := get(ctx)
		if isNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
