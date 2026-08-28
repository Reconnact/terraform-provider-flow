package flow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flowswiss/goclient"
)

var fastPolicy = retryPolicy{
	Timeout:      300 * time.Millisecond,
	InitialDelay: 5 * time.Millisecond,
	MaxDelay:     20 * time.Millisecond,
}

// goclient.APIError cannot be constructed outside its package, so errors are
// produced by a real client against a test server.
func apiServer(t *testing.T, handler http.HandlerFunc) (goclient.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return goclient.NewClient(goclient.WithBase(srv.URL + "/")), srv
}

func jsonError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"message":{"en":"` + message + `"}}}`))
}

func TestRetry_SucceedsAfterTransientErrors(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			jsonError(w, http.StatusBadRequest, "Router still serves instances with attached elastic ips")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := retryWith(context.Background(), fastPolicy, opMutate, "delete router interface", func() error {
		return client.Delete(context.Background(), "/v4/compute/routers/1/interfaces/1")
	})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_GivesUpAfterBudgetWithOriginalError(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		jsonError(w, http.StatusBadRequest, "Unable to detach root volume")
	})

	start := time.Now()
	err := retryWith(context.Background(), fastPolicy, opMutate, "detach volume", func() error {
		return client.Delete(context.Background(), "/v4/compute/volumes/1/instances/1")
	})
	if err == nil {
		t.Fatal("expected the persistent error to be returned")
	}
	if got := err.Error(); !containsString(got, "Unable to detach root volume") {
		t.Fatalf("expected the original API message to be preserved, got %q", got)
	}
	if calls < 2 {
		t.Fatalf("expected several attempts, got %d", calls)
	}
	if elapsed := time.Since(start); elapsed > 2*fastPolicy.Timeout {
		t.Fatalf("retry loop overran its budget: %s", elapsed)
	}
}

func TestRetry_DoesNotRetryAuthErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		var calls int32
		client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			jsonError(w, status, "Access denied")
		})

		err := retryWith(context.Background(), fastPolicy, opMutate, "update", func() error {
			return client.Update(context.Background(), "/v4/compute/networks/1", map[string]string{"name": "x"}, nil)
		})
		if err == nil {
			t.Fatalf("status %d: expected an error", status)
		}
		if calls != 1 {
			t.Fatalf("status %d: expected exactly 1 call, got %d", status, calls)
		}
	}
}

func TestRetry_NotFoundOnMutateIsSuccess(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		jsonError(w, http.StatusNotFound, "Router interface not found")
	})

	err := retryWith(context.Background(), fastPolicy, opMutate, "delete router interface", func() error {
		return client.Delete(context.Background(), "/v4/compute/routers/1/interfaces/1")
	})
	if err != nil {
		t.Fatalf("404 on a delete must count as success, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func TestRetry_NotFoundOnCreateIsHardError(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		jsonError(w, http.StatusNotFound, "Desired kubernetes version 0 not found")
	})

	err := retryWith(context.Background(), fastPolicy, opCreate, "update cluster configuration", func() error {
		return client.Set(context.Background(), "/v4/kubernetes/clusters/1/configuration", map[string]int{"version_id": 0}, nil)
	})
	if err == nil {
		t.Fatal("expected the 404 to be returned")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func TestRetry_ServerErrorsAreRetried(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			jsonError(w, http.StatusInternalServerError, "An error occurred")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":263}`))
	})

	var out struct {
		ID int `json:"id"`
	}
	err := retryWith(context.Background(), fastPolicy, opCreate, "create elastic ip", func() error {
		return client.Create(context.Background(), "/v4/compute/elastic-ips", map[string]int{"location_id": 1}, &out)
	})
	if err != nil {
		t.Fatalf("expected success after the 500, got %v", err)
	}
	if out.ID != 263 || calls != 2 {
		t.Fatalf("expected id 263 after 2 calls, got id %d after %d calls", out.ID, calls)
	}
}

func TestRetry_UnparseableErrorBodyIsRetried(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := retryWith(context.Background(), fastPolicy, opCreate, "create server", func() error {
		return client.Create(context.Background(), "/v4/compute/instances", map[string]string{"name": "x"}, nil)
	})
	if err != nil {
		t.Fatalf("expected success after the gateway error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestRetry_TransportErrorRetriedForMutateOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	client := goclient.NewClient(goclient.WithBase(url + "/"))

	var mutateCalls int32
	err := retryWith(context.Background(), fastPolicy, opMutate, "detach", func() error {
		atomic.AddInt32(&mutateCalls, 1)
		return client.Delete(context.Background(), "/v4/compute/volumes/1/instances/1")
	})
	if err == nil {
		t.Fatal("expected the transport error to surface after the budget")
	}
	if mutateCalls < 2 {
		t.Fatalf("mutate: expected several attempts on a transport error, got %d", mutateCalls)
	}

	var createCalls int32
	err = retryWith(context.Background(), fastPolicy, opCreate, "create", func() error {
		atomic.AddInt32(&createCalls, 1)
		return client.Create(context.Background(), "/v4/compute/instances", map[string]string{"name": "x"}, nil)
	})
	if err == nil {
		t.Fatal("expected the transport error to surface")
	}
	if createCalls != 1 {
		t.Fatalf("create: a transport error must not be retried (duplicate risk), got %d calls", createCalls)
	}
}

func TestRetry_StopsWhenContextIsCancelled(t *testing.T) {
	var calls int32
	client, _ := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		jsonError(w, http.StatusBadRequest, "An error occurred")
	})

	ctx, cancel := context.WithCancel(context.Background())
	slow := retryPolicy{Timeout: 10 * time.Second, InitialDelay: 200 * time.Millisecond, MaxDelay: time.Second}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := retryWith(ctx, slow, opMutate, "delete", func() error {
		return client.Delete(ctx, "/v4/compute/networks/1")
	})
	if err == nil {
		t.Fatal("expected an error after cancellation")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("retry loop did not stop promptly on cancellation")
	}
}

func TestRetry_NoRetryOnCallerContextErrors(t *testing.T) {
	var calls int32
	err := retryWith(context.Background(), fastPolicy, opMutate, "x", func() error {
		atomic.AddInt32(&calls, 1)
		return context.DeadlineExceeded
	})
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("expected deadline error after 1 call, got %v after %d calls", err, calls)
	}
}

func TestParseRetryTimeout(t *testing.T) {
	if d, err := parseRetryTimeout("2m"); err != nil || d != 2*time.Minute {
		t.Fatalf("2m: got %s, %v", d, err)
	}
	if _, err := parseRetryTimeout("soon"); err == nil {
		t.Fatal("expected an error for an invalid duration")
	}
	if _, err := parseRetryTimeout("-5s"); err == nil {
		t.Fatal("expected an error for a negative duration")
	}
	if d, err := parseRetryTimeout("0"); err != nil || d != 0 {
		t.Fatalf("0 (retries disabled): got %s, %v", d, err)
	}
}

func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
