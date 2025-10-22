package middleware

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mainuli/artifusion/internal/errors"
)

// Timeout Middleware Design Notes
//
// CLIENT DISCONNECTION DETECTION:
//
// This middleware does NOT implement http.CloseNotifier (deprecated since Go 1.7).
// This is a deliberate architectural decision for a modern Go codebase.
//
// WHY CloseNotifier IS NOT SUPPORTED:
//
//  1. Deprecated for 9 years - CloseNotifier was deprecated when Go 1.7 introduced
//     Request.Context() in 2016. It's legacy compatibility code in the stdlib.
//
//  2. Context is superior - Since Go 1.8 (2017), Request.Context() automatically
//     cancels when the client disconnects, providing a cleaner API than CloseNotifier.
//
//  3. This is an unreleased project - We can build with modern patterns from day 1
//     without backwards compatibility concerns.
//
//  4. No current handlers need it - All handlers in this codebase are request-response
//     proxies that don't require client disconnection detection.
//
// HOW TO DETECT CLIENT DISCONNECTION (Modern Approach):
//
// Handlers that need to detect client disconnection should use Request.Context():
//
//	func longRunningHandler(w http.ResponseWriter, r *http.Request) {
//	    ctx := r.Context()
//
//	    // Start long-running work
//	    resultCh := make(chan Result)
//	    go doWork(ctx, resultCh)
//
//	    // Wait for either completion or cancellation
//	    select {
//	    case <-ctx.Done():
//	        // Client disconnected, timeout occurred, or request cancelled
//	        // Context provides the reason via ctx.Err():
//	        //   - context.DeadlineExceeded (timeout)
//	        //   - context.Canceled (client disconnect or explicit cancel)
//	        log.Printf("Request cancelled: %v", ctx.Err())
//	        return
//
//	    case result := <-resultCh:
//	        // Work completed successfully
//	        w.Write(result.Data)
//	    }
//	}
//
// WHEN CONTEXT IS CANCELLED:
//
// The Request.Context() is automatically cancelled when:
//   - Client closes the connection (TCP disconnect)
//   - HTTP/2 RST_STREAM received (HTTP/2 cancellation)
//   - ServeHTTP method returns
//   - Timeout middleware deadline exceeded (via context.WithTimeout)
//
// REFERENCES:
//
//   - Go 1.7 Release Notes (2016): https://go.dev/doc/go1.7#net_http
//     "The Request.Context method returns a context for the request..."
//
//   - Go 1.8 Release Notes (2017): https://go.dev/doc/go1.8#net_http
//     "The Server now supports graceful shutdown... the provided Context is
//     canceled when the client's connection closes..."
//
//   - CloseNotifier deprecation: https://pkg.go.dev/net/http#CloseNotifier
//     "Deprecated: the CloseNotifier interface predates Go's context package.
//     New code should use Request.Context instead."
//
// BACKWARDS COMPATIBILITY - SYNTHETIC CloseNotifier:
//
// While we don't pass through http.CloseNotifier from the underlying ResponseWriter
// (it's deprecated), we DO provide a synthetic CloseNotifier implementation that
// bridges to Request.Context().
//
// This means legacy handlers using CloseNotify() will work correctly:
//
//	func legacyHandler(w http.ResponseWriter, r *http.Request) {
//	    notify := w.(http.CloseNotifier).CloseNotify()
//	    select {
//	    case <-notify:
//	        log.Println("Client disconnected")
//	        return
//	    case <-time.After(10 * time.Second):
//	        w.Write([]byte("Done"))
//	    }
//	}
//
// How it works:
//  1. ResponseWriter implements CloseNotifier (our synthetic implementation)
//  2. CloseNotify() returns a channel that fires when ctx.Done() fires
//  3. Context is cancelled automatically on client disconnect (Go 1.8+)
//  4. Goroutine spawned only when CloseNotify() is actually called (lazy)
//
// Benefits:
//   - No deprecation warnings (we implement the interface, not consume it)
//   - Works everywhere (not dependent on underlying ResponseWriter)
//   - Uses context as single source of truth
//   - Backwards compatible with legacy code
//
// timeoutWriter is the core ResponseWriter wrapper. It isolates headers so we
// can safely send a timeout response from the main goroutine while the handler
// may still be running, and it drops writes once the timeout is triggered.
type timeoutWriter struct {
	w           http.ResponseWriter
	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
	header      http.Header
}

const (
	capabilityFlusher = 1 << iota
	capabilityHijacker
	capabilityPusher
)

func newTimeoutWriter(w http.ResponseWriter) (http.ResponseWriter, *timeoutWriter) {
	tw := &timeoutWriter{
		w:      w,
		header: make(http.Header),
	}

	var (
		caps int
		fl   http.Flusher
		hj   http.Hijacker
		ps   http.Pusher
	)

	if v, ok := w.(http.Flusher); ok {
		caps |= capabilityFlusher
		fl = v
	}
	if v, ok := w.(http.Hijacker); ok {
		caps |= capabilityHijacker
		hj = v
	}
	if v, ok := w.(http.Pusher); ok {
		caps |= capabilityPusher
		ps = v
	}

	return wrapTimeoutWriter(caps, tw, fl, hj, ps), tw
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.header
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}

	if !tw.wroteHeader {
		tw.writeHeaderLocked(http.StatusOK)
	}

	return tw.w.Write(p)
}

func (tw *timeoutWriter) WriteHeader(statusCode int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.wroteHeader {
		return
	}

	tw.writeHeaderLocked(statusCode)
}

// writeHeaderLocked copies the buffered headers onto the underlying writer and
// writes the provided status code. Caller must hold tw.mu.
func (tw *timeoutWriter) writeHeaderLocked(statusCode int) {
	tw.wroteHeader = true

	dst := tw.w.Header()
	for k, vv := range tw.header {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}

	tw.w.WriteHeader(statusCode)
}

// timeout marks the writer as timed out and blocks future writes. It returns
// true if headers were already sent to the client.
func (tw *timeoutWriter) timeout() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.timedOut = true
	return tw.wroteHeader
}

func (tw *timeoutWriter) flush(fl http.Flusher) {
	tw.mu.Lock()
	if tw.timedOut {
		tw.mu.Unlock()
		return
	}
	if !tw.wroteHeader {
		tw.writeHeaderLocked(http.StatusOK)
	}
	tw.mu.Unlock()

	fl.Flush()
}

func (tw *timeoutWriter) hijack(hj http.Hijacker) (net.Conn, *bufio.ReadWriter, error) {
	tw.mu.Lock()
	if tw.timedOut {
		tw.mu.Unlock()
		return nil, nil, http.ErrHandlerTimeout
	}
	tw.wroteHeader = true
	tw.mu.Unlock()

	return hj.Hijack()
}

func (tw *timeoutWriter) push(ps http.Pusher, target string, opts *http.PushOptions) error {
	tw.mu.Lock()
	if tw.timedOut {
		tw.mu.Unlock()
		return http.ErrHandlerTimeout
	}
	tw.mu.Unlock()

	return ps.Push(target, opts)
}

// Mixins for optional interfaces. Each provides the relevant method without
// duplicating the timeout-aware logic.

type flushMixin struct {
	tw      *timeoutWriter
	flusher http.Flusher
}

func (m *flushMixin) Flush() {
	m.tw.flush(m.flusher)
}

type hijackMixin struct {
	tw       *timeoutWriter
	hijacker http.Hijacker
}

func (m *hijackMixin) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return m.tw.hijack(m.hijacker)
}

type pushMixin struct {
	tw     *timeoutWriter
	pusher http.Pusher
}

func (m *pushMixin) Push(target string, opts *http.PushOptions) error {
	return m.tw.push(m.pusher, target, opts)
}

// Wrapper types for combinations of optional interfaces. They embed the core
// timeoutWriter (directly or via another wrapper) along with the mixins that
// supply additional methods.

type writerF struct {
	*timeoutWriter
	flushMixin
}

type writerH struct {
	*timeoutWriter
	hijackMixin
}

type writerP struct {
	*timeoutWriter
	pushMixin
}

type writerFH struct {
	*writerF
	hijackMixin
}

type writerFP struct {
	*writerF
	pushMixin
}

type writerHP struct {
	*writerH
	pushMixin
}

type writerFHP struct {
	*writerFH
	pushMixin
}

func wrapTimeoutWriter(caps int, tw *timeoutWriter, fl http.Flusher, hj http.Hijacker, ps http.Pusher) http.ResponseWriter {
	switch caps {
	case 0:
		return tw
	case capabilityFlusher:
		return &writerF{
			timeoutWriter: tw,
			flushMixin:    flushMixin{tw: tw, flusher: fl},
		}
	case capabilityHijacker:
		return &writerH{
			timeoutWriter: tw,
			hijackMixin:   hijackMixin{tw: tw, hijacker: hj},
		}
	case capabilityPusher:
		return &writerP{
			timeoutWriter: tw,
			pushMixin:     pushMixin{tw: tw, pusher: ps},
		}
	case capabilityFlusher | capabilityHijacker:
		flWriter := &writerF{
			timeoutWriter: tw,
			flushMixin:    flushMixin{tw: tw, flusher: fl},
		}
		return &writerFH{
			writerF: flWriter,
			hijackMixin: hijackMixin{
				tw:       tw,
				hijacker: hj,
			},
		}
	case capabilityFlusher | capabilityPusher:
		flWriter := &writerF{
			timeoutWriter: tw,
			flushMixin:    flushMixin{tw: tw, flusher: fl},
		}
		return &writerFP{
			writerF: flWriter,
			pushMixin: pushMixin{
				tw:     tw,
				pusher: ps,
			},
		}
	case capabilityHijacker | capabilityPusher:
		hjWriter := &writerH{
			timeoutWriter: tw,
			hijackMixin:   hijackMixin{tw: tw, hijacker: hj},
		}
		return &writerHP{
			writerH: hjWriter,
			pushMixin: pushMixin{
				tw:     tw,
				pusher: ps,
			},
		}
	case capabilityFlusher | capabilityHijacker | capabilityPusher:
		fhWriter := &writerFH{
			writerF: &writerF{
				timeoutWriter: tw,
				flushMixin:    flushMixin{tw: tw, flusher: fl},
			},
			hijackMixin: hijackMixin{
				tw:       tw,
				hijacker: hj,
			},
		}
		return &writerFHP{
			writerFH: fhWriter,
			pushMixin: pushMixin{
				tw:     tw,
				pusher: ps,
			},
		}
	default:
		return tw
	}
}

// contextCloseNotifier wraps a ResponseWriter and provides CloseNotifier by
// watching the request context. This bridges modern context-based cancellation
// to the deprecated CloseNotifier interface for backwards compatibility.
//
// DESIGN: This is a synthetic implementation that doesn't use the deprecated
// http.CloseNotifier from the underlying ResponseWriter. Instead, it watches
// Request.Context().Done() and sends a signal when the context is cancelled.
//
// This approach:
//   - Generates no deprecation warnings (we implement, not consume)
//   - Works even if underlying ResponseWriter lacks CloseNotifier
//   - Uses context as single source of truth for cancellation
//   - Spawns goroutine only when CloseNotify() is actually called (lazy)
type contextCloseNotifier struct {
	http.ResponseWriter
	closeCh chan bool
	ctx     context.Context
	once    sync.Once
}

// CloseNotify returns a channel that receives a single value when the client
// connection has gone away. This is implemented by watching ctx.Done().
func (w *contextCloseNotifier) CloseNotify() <-chan bool {
	w.once.Do(func() {
		// Start goroutine to watch context (lazy - only when CloseNotify() is called)
		go func() {
			<-w.ctx.Done()
			// Send signal (non-blocking, channel is buffered)
			select {
			case w.closeCh <- true:
			default:
			}
		}()
	})
	return w.closeCh
}

// Preserve other ResponseWriter interfaces by delegating to wrapped writer
// This ensures Flush, Hijack, Push still work if the underlying writer supports them

func (w *contextCloseNotifier) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *contextCloseNotifier) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (w *contextCloseNotifier) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Timeout enforces a maximum duration on every request. Once the deadline
// passes we send a timeout response (if possible) and drop further writes from
// the handler to keep the underlying ResponseWriter safe.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			done := make(chan struct{})
			panicChan := make(chan interface{}, 1)

			wrapped, core := newTimeoutWriter(w)

			// COMPATIBILITY: Add synthetic CloseNotifier support
			// This bridges Request.Context() to the deprecated CloseNotify() interface
			// for legacy handlers that may still rely on it.
			//
			// Only add if context has a Done channel (i.e., not Background context).
			// The Background context has a nil Done channel and won't benefit from this.
			if ctx.Done() != nil {
				wrapped = &contextCloseNotifier{
					ResponseWriter: wrapped,
					closeCh:        make(chan bool, 1), // Buffered to prevent goroutine leak
					ctx:            ctx,
				}
			}

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()

				next.ServeHTTP(wrapped, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case p := <-panicChan:
				panic(p)
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					headersWritten := core.timeout()
					if !headersWritten {
						errors.ErrorResponse(w, errors.ErrBackendTimeout.WithMessage(
							"Request exceeded maximum allowed duration"))
					}
				}
			}
		})
	}
}
