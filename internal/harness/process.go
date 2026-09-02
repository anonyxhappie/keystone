package harness

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"sync"
	"time"
)

// ProcessAdapter is a generic Level-3/4 harness boundary. It deliberately
// knows nothing about a provider protocol: the configured process is external.
type ProcessAdapter struct { Command string; Args []string; cmd *exec.Cmd; stdin io.WriteCloser; out *bufio.Reader; mu sync.Mutex }
type ProcessEvent struct { Type string; Data string; At time.Time }

func (a *ProcessAdapter) Start(ctx context.Context) error {
	a.mu.Lock(); defer a.mu.Unlock(); a.cmd=exec.CommandContext(ctx,a.Command,a.Args...); in,err:=a.cmd.StdinPipe();if err!=nil{return err};out,err:=a.cmd.StdoutPipe();if err!=nil{return err};a.cmd.Stderr=a.cmd.Stdout;a.stdin=in;a.out=bufio.NewReader(out);return a.cmd.Start()
}
func (a *ProcessAdapter) Send(s string) error { a.mu.Lock(); defer a.mu.Unlock(); if a.stdin==nil{return io.ErrClosedPipe};_,err:=io.WriteString(a.stdin,s+"\n");return err }
func (a *ProcessAdapter) Observe() (ProcessEvent,error) { a.mu.Lock(); defer a.mu.Unlock(); s,err:=a.out.ReadString('\n');if err!=nil{return ProcessEvent{},err};return ProcessEvent{Type:"message",Data:s,At:time.Now().UTC()},nil }
func (a *ProcessAdapter) Interrupt() error { a.mu.Lock(); defer a.mu.Unlock(); if a.cmd==nil||a.cmd.Process==nil{return nil};return a.cmd.Process.Kill() }
func (a *ProcessAdapter) Wait() error { a.mu.Lock(); c:=a.cmd;a.mu.Unlock();if c==nil{return nil};return c.Wait() }
