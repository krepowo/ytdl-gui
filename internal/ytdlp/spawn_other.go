//go:build !windows

package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// spawnProcess is a non-Windows stub so the package still builds and its
// lifecycle tests run on other platforms. The shipped app targets Windows only;
// pause is unsupported here and falls back to kill + --continue.
func spawnProcess(ctx context.Context, name string, args []string) (child, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("ytdlp: create pipe: %w", err)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		return nil, fmt.Errorf("ytdlp: start %s: %w", name, err)
	}
	_ = w.Close()

	return &posixChild{cmd: cmd, r: r}, nil
}

type posixChild struct {
	cmd *exec.Cmd
	r   *os.File

	mu     sync.Mutex
	waited bool
	err    error
}

func (c *posixChild) stdout() io.ReadCloser { return c.r }

func (c *posixChild) wait() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.waited {
		return c.err
	}
	c.waited = true
	c.err = c.cmd.Wait()
	return c.err
}

// suspend is unsupported off Windows; the job falls back to kill + --continue.
func (c *posixChild) suspend() error { return errors.New("ytdlp: suspend unsupported") }

func (c *posixChild) resume() error { return errors.New("ytdlp: resume unsupported") }

func (c *posixChild) kill() error {
	if c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

func (c *posixChild) close() {
	if c.r != nil {
		_ = c.r.Close()
	}
}
