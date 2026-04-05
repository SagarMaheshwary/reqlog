package docker

import (
	"io"
	"os/exec"
)

type CMDReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *CMDReadCloser) Close() error {
	err := c.ReadCloser.Close()
	waitErr := c.cmd.Wait()

	if err != nil {
		return err
	}
	return waitErr
}
