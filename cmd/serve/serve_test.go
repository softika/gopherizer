//go:build unix

package serve

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// TestNotifyShutdownCancelsOnSigterm covers the signal a container runtime
// actually sends: `docker stop` and Kubernetes both deliver SIGTERM.
func TestNotifyShutdownCancelsOnSigterm(t *testing.T) {
	ctx, stop := notifyShutdown()
	defer stop()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("context was not cancelled on SIGTERM; graceful shutdown would never run under docker/k8s")
	}
}

func TestNotifyShutdownCancelsOnSigint(t *testing.T) {
	ctx, stop := notifyShutdown()
	defer stop()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("context was not cancelled on SIGINT")
	}
}

// TestShutdownSignalsExcludeSigkill documents intent: SIGKILL can never be
// caught or handled, so registering it is misleading dead configuration.
func TestShutdownSignalsExcludeSigkill(t *testing.T) {
	for _, sig := range shutdownSignals {
		if sig == os.Kill || sig == syscall.SIGKILL {
			t.Fatalf("SIGKILL cannot be caught or handled; it must not be registered")
		}
	}
}
