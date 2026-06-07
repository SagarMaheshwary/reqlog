package transport

import (
	"io"
	"os/exec"
)

type LocalExecutor struct{}

func (e *LocalExecutor) Run(name string, args ...string) (io.ReadCloser, error) {
	cmd := exec.Command(name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &LocalReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
	}, nil
}

func (e *LocalExecutor) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type LocalReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *LocalReadCloser) Close() error {
	err := c.ReadCloser.Close()
	waitErr := c.cmd.Wait()

	if err != nil {
		return err
	}
	return waitErr
}
