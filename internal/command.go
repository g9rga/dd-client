package internal

import (
	"bufio"
	"context"
	"os/exec"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

type CommandPool struct {
	sync.RWMutex
	commands map[string]*exec.Cmd
}

func (c *CommandPool) UpsertCommand() error {
	return nil
}

func CreateCommandPool() *CommandPool {
	return &CommandPool{
		commands: make(map[string]*exec.Cmd),
	}
}

func (c *CommandPool) GetCommands() map[string]*exec.Cmd {
	return c.commands
}
func (c *CommandPool) StopCommand(cid string) error {
	c.Lock()
	defer c.Unlock()
	err := c.commands[cid].Process.Signal(syscall.SIGKILL)
	delete(c.commands, cid)
	return err
}
func (c *CommandPool) RunCommand(ctx context.Context, cid string, command string, args []string) error {
	c.Lock()
	entry := logrus.WithField("ctx", map[string]interface{}{
		"cid":  cid,
		"args": args,
	})
	if _, ok := c.commands[cid]; ok {
		entry.Info("Already started, skipping")
		c.Unlock()
		return nil
	}
	cmd := exec.CommandContext(ctx, command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		entry.Error(err)
		c.Unlock()
		return err
	}
	stdoutScanner := bufio.NewScanner(stdout)
	go func() {
		for stdoutScanner.Scan() {
			entry.Info(stdoutScanner.Text())
		}
	}()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.Unlock()
		return err
	}
	stderrScanner := bufio.NewScanner(stderr)
	go func() {
		for stderrScanner.Scan() {
			entry.Warning("error: " + stderrScanner.Text())
		}
	}()
	if err = cmd.Start(); err != nil {
		c.Unlock()
		return err
	}
	c.commands[cid] = cmd
	c.Unlock()
	entry.Info("Started")
	_ = cmd.Wait()
	c.Lock()
	defer c.Unlock()
	delete(c.commands, cid)
	entry.Info("Exited")
	return nil
}
