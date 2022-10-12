package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-cmd/cmd"
	"github.com/pkg/errors"
)

var (
	lineBufferSize uint = 1024 * 32
	errNonZero          = errors.New("non-zero exit code")
	errTimeout          = errors.New("timeout for command")
)

type Command struct {
	stdoutFunc, stderrFunc LineFunc
	cfg                    Config
}

type LineFunc func(line string)

var StdoutLineFunc LineFunc = func(line string) {
	if _, err := fmt.Fprintln(os.Stdout, line); err != nil {
		log.Printf("failed to write stdout, error: %+v", err)
	}
}

var StderrLineFunc LineFunc = func(line string) {
	if _, err := fmt.Fprintln(os.Stderr, line); err != nil {
		log.Printf("failed to write stderr, error: %+v", err)
	}
}

func NewCommand(stdoutFunc, stderrFunc LineFunc) *Command {
	if stderrFunc == nil {
		stderrFunc = StderrLineFunc
	}
	if stdoutFunc == nil {
		stdoutFunc = StdoutLineFunc
	}

	return &Command{
		stdoutFunc: stdoutFunc,
		stderrFunc: stderrFunc,
	}
}

func (h *Command) WithConfig(config Config) error {
	h.cfg = config
	return nil
}

func (h *Command) processOutput(command *cmd.Cmd, chOutputDone chan struct{}) {
	for command.Stdout != nil || command.Stderr != nil {
		select {
		case line, open := <-command.Stdout:
			if !open {
				command.Stdout = nil
				continue
			}
			h.stdoutFunc(line)
		case line, open := <-command.Stderr:
			if !open {
				command.Stderr = nil
				continue
			}
			h.stderrFunc(line)
		}
	}

	chOutputDone <- struct{}{}
}

func (h *Command) Run(
	ctx context.Context,
	program string,
	args []string,
	envs []string,
) error {
	command := cmd.NewCmdOptions(cmd.Options{
		Buffered:       false,
		Streaming:      true,
		LineBufferSize: lineBufferSize,
	}, program, args...)

	command.Env = os.Environ()
	command.Env = append(command.Env, envs...)
	statusCh := command.Start()
	chOutputDone := make(chan struct{}, 1)

	stopCommand := func() {
		if cerr := command.Stop(); cerr != nil && !errors.Is(cerr, cmd.ErrNotStarted) {
			log.Printf("failed stopping exec, error: %v", cerr)
		}
	}
	defer func() {
		<-chOutputDone
		stopCommand()
	}()

	go h.processOutput(command, chOutputDone)

	select {
	case status := <-statusCh:
		if status.Error != nil {
			return errors.Wrapf(status.Error, "failed to execute command, program: %s", program)
		}

		if status.Complete {
			if status.Exit != 0 {
				return errors.Wrapf(
					errNonZero,
					"failed to execute command, with non-zero exit code: %d, program: %s",
					status.Exit,
					program,
				)
			}
			return nil
		}

	case <-ctx.Done():
		log.Println("Terminated by signal")
		stopCommand()
		return errTimeout
	}

	return nil
}

func Run(ctx context.Context, program string, args []string) error {
	return NewCommand(StdoutLineFunc, StderrLineFunc).Run(ctx, program, args, nil)
}
