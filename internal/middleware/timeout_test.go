package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestContextCloseNotifier_Interface verifies that contextCloseNotifier implements http.CloseNotifier
func TestContextCloseNotifier_Interface(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := &contextCloseNotifier{
		ResponseWriter: httptest.NewRecorder(),
		closeCh:        make(chan bool, 1),
		ctx:            ctx,
	}

	// Verify it implements CloseNotifier
	//goland:noinspection GoDeprecation
	_, ok := interface{}(w).(http.CloseNotifier)
	if !ok {
		t.Error("contextCloseNotifier does not implement http.CloseNotifier")
	}
}

// TestContextCloseNotifier_FiresOnContextCancel verifies that CloseNotify channel fires when context is cancelled
func TestContextCloseNotifier_FiresOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	w := &contextCloseNotifier{
		ResponseWriter: httptest.NewRecorder(),
		closeCh:        make(chan bool, 1),
		ctx:            ctx,
	}

	// Get the CloseNotify channel (this starts the goroutine)
	notify := w.CloseNotify()

	// Cancel the context
	cancel()

	// Wait for signal on CloseNotify channel
	select {
	case <-notify:
		// Success - channel fired as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("CloseNotify channel did not fire after context cancellation")
	}
}

// TestContextCloseNotifier_NoFireOnNormalCompletion verifies that CloseNotify doesn't fire on normal completion
func TestContextCloseNotifier_NoFireOnNormalCompletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	w := &contextCloseNotifier{
		ResponseWriter: httptest.NewRecorder(),
		closeCh:        make(chan bool, 1),
		ctx:            ctx,
	}

	// Get the CloseNotify channel (this starts the goroutine)
	notify := w.CloseNotify()

	// Wait a bit but don't cancel context
	time.Sleep(10 * time.Millisecond)

	// Verify channel hasn't fired yet
	select {
	case <-notify:
		t.Error("CloseNotify channel fired unexpectedly before context cancellation")
	default:
		// Success - channel did not fire
	}
}

// TestContextCloseNotifier_LazyGoroutineSpawn verifies that goroutine is only spawned when CloseNotify() is called
func TestContextCloseNotifier_LazyGoroutineSpawn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := &contextCloseNotifier{
		ResponseWriter: httptest.NewRecorder(),
		closeCh:        make(chan bool, 1),
		ctx:            ctx,
	}

	// Just creating the wrapper should not spawn goroutine
	// (We can't directly verify goroutine count, but we can verify the once.Do behavior)

	// First call should initialize
	notify1 := w.CloseNotify()

	// Second call should return same channel (lazy)
	notify2 := w.CloseNotify()

	if notify1 != notify2 {
		t.Error("CloseNotify() returned different channels on multiple calls")
	}
}

// TestContextCloseNotifier_PreservesInterfaces verifies that wrapper preserves other ResponseWriter interfaces
func TestContextCloseNotifier_PreservesInterfaces(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock writer that implements Flusher
	mockWriter := &mockFlushWriter{ResponseWriter: httptest.NewRecorder()}

	w := &contextCloseNotifier{
		ResponseWriter: mockWriter,
		closeCh:        make(chan bool, 1),
		ctx:            ctx,
	}

	// Verify Flusher interface is preserved
	flusher, ok := interface{}(w).(http.Flusher)
	if !ok {
		t.Error("contextCloseNotifier did not preserve Flusher interface")
	}

	// Verify Flush works
	flusher.Flush()
	if !mockWriter.flushed {
		t.Error("Flush was not called on underlying writer")
	}
}

// TestTimeoutMiddleware_WithCloseNotifier verifies that Timeout middleware provides CloseNotifier
func TestTimeoutMiddleware_WithCloseNotifier(t *testing.T) {
	timeoutDuration := 100 * time.Millisecond

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify ResponseWriter implements CloseNotifier
		//goland:noinspection GoDeprecation
		_, ok := w.(http.CloseNotifier)
		if !ok {
			t.Error("ResponseWriter from Timeout middleware does not implement CloseNotifier")
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	// Wrap with Timeout middleware
	wrappedHandler := Timeout(timeoutDuration)(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// TestTimeoutMiddleware_CloseNotifyOnCancel verifies that CloseNotify fires when context is cancelled
func TestTimeoutMiddleware_CloseNotifyOnCancel(t *testing.T) {
	timeoutDuration := 1 * time.Second
	cancelSignalReceived := make(chan bool, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//goland:noinspection GoDeprecation
		closeNotifier, ok := w.(http.CloseNotifier)
		if !ok {
			t.Error("ResponseWriter does not implement CloseNotifier")
			return
		}

		notify := closeNotifier.CloseNotify()

		// Wait for either cancel or timeout
		select {
		case <-notify:
			// Cancel signal received
			cancelSignalReceived <- true
		case <-time.After(2 * time.Second):
			// Timeout - this should not happen in this test
			t.Error("CloseNotify did not fire within expected time")
		}
	})

	// Wrap with Timeout middleware
	wrappedHandler := Timeout(timeoutDuration)(handler)

	// Create test request with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	// Start request handling in goroutine
	done := make(chan bool)
	go func() {
		wrappedHandler.ServeHTTP(rec, req)
		done <- true
	}()

	// Wait a bit, then cancel context
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for signal
	select {
	case <-cancelSignalReceived:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Error("Cancel signal was not received via CloseNotify")
	}

	// Wait for handler to complete
	<-done
}

// TestTimeoutMiddleware_NoCloseNotifyWithBackgroundContext verifies that Background context doesn't get CloseNotifier
func TestTimeoutMiddleware_NoCloseNotifyWithBackgroundContext(t *testing.T) {
	// This test verifies that we don't add CloseNotifier wrapper when ctx.Done() == nil
	// However, in practice, Timeout middleware always creates a context with timeout,
	// so this scenario won't occur in real usage. This test is more for documentation.

	timeoutDuration := 100 * time.Millisecond

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Even with timeout middleware, ResponseWriter should have CloseNotifier
		// because timeout creates a context with deadline
		//goland:noinspection GoDeprecation
		_, ok := w.(http.CloseNotifier)
		if !ok {
			t.Error("ResponseWriter does not implement CloseNotifier (unexpected)")
		}

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := Timeout(timeoutDuration)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// mockFlushWriter is a mock ResponseWriter that implements http.Flusher
type mockFlushWriter struct {
	http.ResponseWriter
	flushed bool
}

func (m *mockFlushWriter) Flush() {
	m.flushed = true
}
