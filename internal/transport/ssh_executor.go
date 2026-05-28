package transport

import (
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

type SSHExecutor struct {
	client *ssh.Client
}

func (e *SSHExecutor) Run(name string, args ...string) (io.ReadCloser, error) {
	session, err := e.client.NewSession()
	if err != nil {
		return nil, err
	}

	cmd := strings.Join(append([]string{name}, args...), " ")

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			io.Copy(pw, stdout)
		}()

		go func() {
			defer wg.Done()
			io.Copy(pw, stderr)
		}()

		wg.Wait()
	}()

	return pr, nil
}

func (e *SSHExecutor) RunCombined(name string, args ...string) (io.ReadCloser, error) {
	return e.Run(name, append(args, "2>&1")...)
}

func (e *SSHExecutor) Output(name string, args ...string) ([]byte, error) {
	session, err := e.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	cmd := strings.Join(append([]string{name}, args...), " ")

	return session.Output(cmd)
}

type SSHReadCloser struct {
	io.Reader
	session *ssh.Session
}

func (r *SSHReadCloser) Close() error {
	waitErr := r.session.Wait()
	closeErr := r.session.Close()

	if waitErr != nil {
		return waitErr
	}

	return closeErr
}
