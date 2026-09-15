package flow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// the `timeouts {}` block as the framework hands it to a resource: null when
// the user left it out, otherwise the durations they wrote
func timeoutsValue(t *testing.T, create string) timeouts.Value {
	t.Helper()

	attrTypes := map[string]attr.Type{"create": types.StringType}
	if create == "" {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}

	object, diags := types.ObjectValue(attrTypes, map[string]attr.Value{"create": types.StringValue(create)})
	if diags.HasError() {
		t.Fatalf("building the timeouts value: %v", diags)
	}
	return timeouts.Value{Object: object}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name      string
		configure string
		want      time.Duration
		wantError bool
	}{
		{name: "unset leaves the context alone", configure: ""},
		{name: "configured sets the deadline", configure: "45m", want: 45 * time.Minute},
		{name: "unparseable is reported", configure: "soon", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics

			ctx, cancel := withTimeout(context.Background(), timeoutsValue(t, test.configure).Create, &diagnostics)
			defer cancel()

			if got := diagnostics.HasError(); got != test.wantError {
				t.Fatalf("diagnostics error = %t, want %t (%v)", got, test.wantError, diagnostics)
			}

			deadline, ok := ctx.Deadline()
			if test.want == 0 {
				if ok {
					t.Fatalf("context carries a deadline of %s, want none", time.Until(deadline))
				}
				return
			}
			if !ok {
				t.Fatal("context carries no deadline")
			}
			if left := time.Until(deadline); left > test.want || left < test.want-time.Minute {
				t.Errorf("deadline in %s, want about %s", left, test.want)
			}
		})
	}
}

// the configured budget wins over a wait's own default in both directions:
// it shortens a long default and lengthens a short one
func TestRemaining(t *testing.T) {
	if got := remaining(context.Background(), time.Minute); got != time.Minute {
		t.Errorf("without a deadline: %s, want the fallback %s", got, time.Minute)
	}

	for _, budget := range []time.Duration{30 * time.Second, 10 * time.Minute} {
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		got := remaining(ctx, time.Minute)
		cancel()

		if got > budget || got < budget-time.Second {
			t.Errorf("with a %s budget: %s, want about %s", budget, got, budget)
		}
	}
}

// a wait never outlives the operation's budget, however generous its own
// default is, and the error says how long it actually waited
func TestWaitForStopsAtTheContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := waitFor(ctx, time.Hour, 10*time.Millisecond, "the sky to fall", func(ctx context.Context) (bool, error) {
		return false, nil
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("wait succeeded, want a timeout")
	}
	if elapsed > time.Second {
		t.Errorf("waited %s, want the 50ms budget", elapsed)
	}
	if !strings.Contains(err.Error(), "waiting for the sky to fall") {
		t.Errorf("error %q does not name what it waited for", err)
	}
}

// without a budget the wait keeps its own default, and the last error from the
// check survives into the timeout
func TestWaitForReportsTheLastError(t *testing.T) {
	err := waitFor(context.Background(), 20*time.Millisecond, 10*time.Millisecond, "a volume to settle", func(ctx context.Context) (bool, error) {
		return false, errStub
	})

	if err == nil {
		t.Fatal("wait succeeded, want a timeout")
	}
	if !strings.Contains(err.Error(), errStub.Error()) {
		t.Errorf("error %q drops the last error from the check", err)
	}
}

func TestWaitForSucceeds(t *testing.T) {
	calls := 0
	err := waitFor(context.Background(), time.Minute, time.Millisecond, "a server to boot", func(ctx context.Context) (bool, error) {
		calls++
		return calls == 3, nil
	})

	if err != nil {
		t.Fatalf("wait failed: %s", err)
	}
	if calls != 3 {
		t.Errorf("checked %d times, want 3", calls)
	}
}

var errStub = stubError("the volume is still working")

type stubError string

func (e stubError) Error() string { return string(e) }
