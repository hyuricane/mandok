package compose

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
)

func doExec(cmd *exec.Cmd) (*bytes.Buffer, error) {
	buff := bytes.NewBuffer(nil)
	buffErr := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = buffErr
	err := cmd.Run()
	if err != nil {
		if buffErr.Len() > 0 {
			err = fmt.Errorf("%s", buffErr.String())
		}
		return nil, err
	}
	return buff, nil
}

func docExecStream(cmd *exec.Cmd) (out chan string, cancel func(), err error) {
	outpipe, err := cmd.StdoutPipe()
	buffErr := bytes.NewBuffer(nil)
	cmd.Stderr = buffErr
	if err := cmd.Start(); err != nil {
		if buffErr.Len() > 0 {
			err = fmt.Errorf("%s", buffErr.String())
		}
		return nil, nil, err
	}
	out = make(chan string, 100)
	go func(p io.ReadCloser) {
		reader := bufio.NewReader(p)
		line, err := reader.ReadString('\n')
		for err == nil {
			out <- line
			line, err = reader.ReadString('\n')
		}
		if err != io.EOF {
			out <- err.Error()
		}
		close(out)
	}(outpipe)
	cancel = func() {
		var err error
		if cmd.Cancel != nil {
			err = cmd.Cancel()
		}
		if cmd.Process != nil {
			err = cmd.Process.Kill()
		}
		if err != nil {
			log.Printf("[ERROR] cancel %v", err)
		}
	}
	return
}
