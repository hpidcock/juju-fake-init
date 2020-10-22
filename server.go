package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Server struct {
	FakeInitServer
}

var once sync.Once
var exitWG sync.WaitGroup
var entrypointProcess *os.Process
var entrypointMutex sync.Mutex

func (s *Server) Start(ctx context.Context, req *StartRequest) (*StartResponse, error) {
	alreadyDone := true
	once.Do(func() {
		alreadyDone = false
	})
	if alreadyDone {
		return nil, grpc.Errorf(codes.AlreadyExists, "already started")
	}

	c := exec.Command(req.Command, req.Args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), req.Env...)
	err := c.Start()
	if err != nil {
		return nil, err
	}

	entrypointMutex.Lock()
	entrypointProcess = c.Process
	entrypointMutex.Unlock()

	cancel := make(chan struct{})
	sc := make(chan os.Signal, 1)
	signal.Notify(sc)
	go func() {
		defer close(sc)
		defer signal.Stop(sc)
		for {
			select {
			case _, _ = <-cancel:
				return
			case sig, ok := <-sc:
				if !ok {
					return
				}
				c.Process.Signal(sig)
			}
		}
	}()

	go func() {
		defer close(cancel)
		err = c.Wait()
		if exit, ok := err.(*exec.ExitError); ok {
			exitWG.Wait()
			os.Exit(exit.ExitCode())
		} else if err != nil {
			log.Fatal(err)
		}
		exitWG.Wait()
		os.Exit(0)
	}()
	return &StartResponse{}, nil
}

func (s *Server) Exec(streams FakeInit_ExecServer) error {
	exitWG.Add(1)
	defer exitWG.Done()

	execReq, err := streams.Recv()
	if err != nil {
		return err
	}
	if execReq.Type != ExecRequestMessageType_EXEC_REQ_EXEC {
		return grpc.Errorf(codes.InvalidArgument, "must send EXEC_REQ_EXEC first")
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	wg := sync.WaitGroup{}
	exitCode := 0
	defer func() {
		wg.Wait()
		streams.Send(&ExecResponse{
			Type:     ExecResponseMessageType_EXEC_RES_EXIT,
			ExitCode: int32(exitCode),
		})
	}()

	c := exec.Command(execReq.Command, execReq.Args...)
	c.Env = append(os.Environ(), execReq.Env...)
	c.Stdin = stdinReader
	c.Stdout = stdoutWriter
	c.Stderr = stderrWriter
	err = c.Start()
	if exit, ok := err.(*exec.Error); ok {
		exitCode = 127
		streams.Send(&ExecResponse{
			Type:   ExecResponseMessageType_EXEC_RES_STDERR,
			Stderr: []byte(exit.Error()),
		})
		return nil
	} else if err != nil {
		return err
	}

	stdinClosed := false

	go func() {
		for {
			req, err := streams.Recv()
			if err == io.EOF {
				if !stdinClosed {
					stdinWriter.Close()
				}
				c.Process.Kill()
				return
			} else if err != nil {
				if !stdinClosed {
					stdinWriter.CloseWithError(err)
				}
				c.Process.Kill()
				return
			}
			switch req.Type {
			case ExecRequestMessageType_EXEC_REQ_SIGNAL:
				c.Process.Signal(syscall.Signal(req.Signal))
			case ExecRequestMessageType_EXEC_REQ_STDIN:
				if stdinClosed {
					continue
				}
				if len(req.Stdin) == 0 {
					stdinWriter.Close()
					stdinClosed = true
					continue
				}
				stdinWriter.Write(req.Stdin)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 512)
		for {
			n, err := stdoutReader.Read(buf)
			if err == io.EOF {
				err = streams.Send(&ExecResponse{
					Type: ExecResponseMessageType_EXEC_RES_STDOUT,
				})
				if err != nil {
					log.Fatal(err)
				}
				return
			} else if err != nil {
				log.Fatal(err)
			}
			if n == 0 {
				continue
			}
			err = streams.Send(&ExecResponse{
				Type:   ExecResponseMessageType_EXEC_RES_STDOUT,
				Stdout: buf[:n],
			})
			if err == io.EOF {
				return
			} else if err != nil {
				log.Fatal(err)
			}
		}
	}()
	defer stdoutWriter.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 512)
		for {
			n, err := stderrReader.Read(buf)
			if err == io.EOF {
				err = streams.Send(&ExecResponse{
					Type: ExecResponseMessageType_EXEC_RES_STDERR,
				})
				if err != nil {
					log.Fatal(err)
				}
				return
			} else if err != nil {
				log.Fatal(err)
			}
			if n == 0 {
				continue
			}
			err = streams.Send(&ExecResponse{
				Type:   ExecResponseMessageType_EXEC_RES_STDERR,
				Stderr: buf[:n],
			})
			if err == io.EOF {
				return
			} else if err != nil {
				log.Fatal(err)
			}
		}
	}()
	defer stderrWriter.Close()

	err = c.Wait()
	if exit, ok := err.(*exec.ExitError); ok {
		exitCode = exit.ExitCode()
	} else if err != nil {
		return err
	}

	return nil
}

func (s *Server) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	exitWG.Add(1)
	defer exitWG.Done()
	return &PingResponse{}, nil
}

func (s *Server) Signal(ctx context.Context, req *SignalRequest) (*SignalRequest, error) {
	exitWG.Add(1)
	defer exitWG.Done()
	entrypointMutex.Lock()
	defer entrypointMutex.Unlock()
	if entrypointProcess == nil {
		return nil, grpc.Errorf(codes.FailedPrecondition, "entrypoint not started")
	}
	err := entrypointProcess.Signal(syscall.Signal(req.Signal))
	if err != nil && err.Error() != "os: process already finished" {
		return nil, err
	}
	return &SignalRequest{}, nil
}

func (s *Server) Status(ctx context.Context, req *StatusRequest) (*StatusResponse, error) {
	exitWG.Add(1)
	defer exitWG.Done()
	entrypointMutex.Lock()
	defer entrypointMutex.Unlock()
	if entrypointProcess != nil {
		return &StatusResponse{
			Status: Status_RUNNING,
			Pid:    int32(entrypointProcess.Pid),
		}, nil
	}
	return &StatusResponse{
		Status: Status_WAITING,
	}, nil
}
