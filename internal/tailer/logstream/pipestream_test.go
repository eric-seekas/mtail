// Copyright 2020 Google Inc. All Rights Reserved.
// This file is available under the Apache license.

package logstream_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/mtail/internal/logline"
	"github.com/google/mtail/internal/tailer/logstream"
	"github.com/google/mtail/internal/testutil"
	"github.com/google/mtail/internal/waker"
	"golang.org/x/sys/unix"
)

func TestPipeStreamReadCompletedBecauseClosed(t *testing.T) {
	var wg sync.WaitGroup

	tmpDir := testutil.TestTempDir(t)

	name := filepath.Join(tmpDir, "fifo")
	testutil.FatalIfErr(t, unix.Mkfifo(name, 0666))

	lines := make(chan *logline.LogLine, 1)
	ctx, cancel := context.WithCancel(context.Background())
	waker := waker.NewTestAlways()

	ps, err := logstream.New(ctx, &wg, waker, name, lines, false)
	testutil.FatalIfErr(t, err)

	f, err := os.OpenFile(name, os.O_WRONLY, os.ModeNamedPipe)
	testutil.FatalIfErr(t, err)
	testutil.WriteString(t, f, "1\n")

	// Pipes need to be closed to signal to the pipeStream to finish up.
	testutil.FatalIfErr(t, f.Close())

	ps.Stop() // no-op for pipes
	wg.Wait()
	close(lines)

	received := testutil.LinesReceived(lines)
	expected := []*logline.LogLine{
		{context.TODO(), name, "1"},
	}
	testutil.ExpectNoDiff(t, expected, received, testutil.IgnoreFields(logline.LogLine{}, "Context"))

	cancel()

	if !ps.IsComplete() {
		t.Errorf("expecting pipestream to be complete because fifo closed")
	}
}

func TestPipeStreamReadCompletedBecauseCancel(t *testing.T) {
	var wg sync.WaitGroup

	tmpDir := testutil.TestTempDir(t)

	name := filepath.Join(tmpDir, "fifo")
	testutil.FatalIfErr(t, unix.Mkfifo(name, 0666))

	lines := make(chan *logline.LogLine, 1)
	ctx, cancel := context.WithCancel(context.Background())
	waker := waker.NewTestAlways()

	ps, err := logstream.New(ctx, &wg, waker, name, lines, false)
	testutil.FatalIfErr(t, err)

	f, err := os.OpenFile(name, os.O_WRONLY, os.ModeNamedPipe)
	testutil.FatalIfErr(t, err)
	testutil.WriteString(t, f, "1\n")

	cancel()
	wg.Wait()
	close(lines)

	received := testutil.LinesReceived(lines)
	expected := []*logline.LogLine{
		{context.TODO(), name, "1"},
	}
	testutil.ExpectNoDiff(t, expected, received, testutil.IgnoreFields(logline.LogLine{}, "Context"))

	if !ps.IsComplete() {
		t.Errorf("expecting pipestream to be complete because cancelled")
	}
}
