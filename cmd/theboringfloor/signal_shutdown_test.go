package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestAwaitSignalShutdownDoesNothingAfterNormalExit(t *testing.T) {
	signals := make(chan os.Signal)
	runDone := make(chan struct{})
	close(runDone)
	called := false
	awaitSignalShutdown(signals, runDone, time.Second, func() { called = true }, func() { called = true }, func(int) { called = true })
	if called {
		t.Fatal("normal unsignalled exit must not invoke shutdown callbacks")
	}
}

func TestAwaitSignalShutdownLetsNormalTeardownFinish(t *testing.T) {
	signals := make(chan os.Signal, 2)
	runDone := make(chan struct{})
	quit := make(chan struct{}, 1)
	fallback := make(chan struct{}, 1)
	exits := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		awaitSignalShutdown(signals, runDone, time.Second, func() { quit <- struct{}{} }, func() { fallback <- struct{}{} }, func(code int) { exits <- code })
		close(done)
	}()

	signals <- os.Interrupt
	<-quit
	close(runDone)
	<-done
	select {
	case <-fallback:
		t.Fatal("normal Bubble Tea teardown must not use fallback cleanup")
	case code := <-exits:
		t.Fatalf("normal Bubble Tea teardown must not exit directly, got %d", code)
	default:
	}
}

func TestAwaitSignalShutdownFallsBackAfterDeadline(t *testing.T) {
	signals := make(chan os.Signal, 1)
	runDone := make(chan struct{})
	quit := make(chan struct{}, 1)
	fallback := make(chan struct{}, 1)
	exits := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		awaitSignalShutdown(signals, runDone, 10*time.Millisecond, func() { quit <- struct{}{} }, func() { fallback <- struct{}{} }, func(code int) { exits <- code })
		close(done)
	}()

	signals <- syscall.SIGTERM
	<-quit
	select {
	case <-fallback:
	case <-time.After(time.Second):
		t.Fatal("timed-out orderly shutdown must run fallback cleanup")
	}
	select {
	case code := <-exits:
		if code != 1 {
			t.Fatalf("fallback exit code = %d, want 1", code)
		}
	case <-time.After(time.Second):
		t.Fatal("timed-out orderly shutdown must exit")
	}
	<-done
}

func TestAwaitSignalShutdownSecondSignalExitsImmediately(t *testing.T) {
	signals := make(chan os.Signal, 2)
	runDone := make(chan struct{})
	quit := make(chan struct{}, 1)
	fallback := make(chan struct{}, 1)
	exits := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		awaitSignalShutdown(signals, runDone, time.Second, func() { quit <- struct{}{} }, func() { fallback <- struct{}{} }, func(code int) { exits <- code })
		close(done)
	}()

	signals <- os.Interrupt
	<-quit
	signals <- syscall.SIGTERM
	select {
	case code := <-exits:
		if code != 1 {
			t.Fatalf("second-signal exit code = %d, want 1", code)
		}
	case <-time.After(time.Second):
		t.Fatal("second signal must exit immediately")
	}
	<-done
	select {
	case <-fallback:
		t.Fatal("second signal before deadline must not start fallback cleanup")
	default:
	}
}

func TestAwaitSignalShutdownSecondSignalInterruptsFallback(t *testing.T) {
	signals := make(chan os.Signal, 2)
	runDone := make(chan struct{})
	quit := make(chan struct{}, 1)
	fallbackStarted := make(chan struct{}, 1)
	releaseFallback := make(chan struct{})
	exits := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		awaitSignalShutdown(signals, runDone, 10*time.Millisecond, func() { quit <- struct{}{} }, func() {
			fallbackStarted <- struct{}{}
			<-releaseFallback
		}, func(code int) { exits <- code })
		close(done)
	}()

	signals <- os.Interrupt
	<-quit
	select {
	case <-fallbackStarted:
	case <-time.After(time.Second):
		t.Fatal("fallback cleanup did not start")
	}
	signals <- syscall.SIGTERM
	select {
	case code := <-exits:
		if code != 1 {
			t.Fatalf("second-signal exit code = %d, want 1", code)
		}
	case <-time.After(time.Second):
		t.Fatal("second signal during fallback must exit immediately")
	}
	<-done
	close(releaseFallback)
}
