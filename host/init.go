package host

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/host/bridge"
	"github.com/mateussouzaweb/lxg/host/dbus"
	"github.com/mateussouzaweb/lxg/host/run"
)

// Init host daemon services in parallel
func Init(ctx *command.Context) error {

	// Context cancelled when SIGINT or SIGTERM is received
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	// Internal context to cancel all services if any fails
	groupCtx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, 8)

	// Link wayland
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := run.LinkWayland(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Link pipewire
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := run.LinkPipewire(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Link pulse native
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := run.LinkPulseNative(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Link X11 zero
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := run.LinkX11Zero(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Link X11 one
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := run.LinkX11One(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Link X11 auth
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := run.LinkX11Auth(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Start D-Bus proxy in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dbus.InitDBus(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Start bridge socket listener in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := bridge.InitSocket(ctx, groupCtx)
		if err != nil {
			errChan <- err
			cancel()
		}
	}()

	// Wait for all services to stop in a separate goroutine
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect first error if any service failed
	var firstErr error
	for err := range errChan {
		if firstErr == nil && !errors.Is(err, context.Canceled) {
			firstErr = err
		}
	}

	return firstErr
}
