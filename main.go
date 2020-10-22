package main

import (
	"bytes"
	context "context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/goterm/term"
	"google.golang.org/grpc"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative init.proto

func main() {
	var f func() (int, error)
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "listen":
		f = cmdListen
	case "exec":
		f = cmdExec
	case "start":
		f = cmdStart
	case "signal":
		f = cmdSignal
	case "status":
		f = cmdStatus
	default:
		f = func() (int, error) {
			fmt.Fprintf(os.Stderr, "usage: %s [listen | exec | start | signal]\n", os.Args[0])
			return 1, nil
		}
	}
	code, err := f()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

func cmdListen() (int, error) {
	addr := ""
	fs := flag.NewFlagSet(fmt.Sprintf("%s listen", os.Args[0]), flag.ExitOnError)
	fs.StringVar(&addr, "socket", "/var/run/container/juju-fake-init.sock", "path to sock file to listen on")
	fs.Parse(os.Args[2:])

	if _, err := os.Stat(addr); os.IsNotExist(err) {
		// Do nothing
	} else if err != nil {
		return 0, err
	} else {
		err := os.Remove(addr)
		if err != nil {
			return 0, err
		}
	}

	l, err := net.Listen("unix", addr)
	if err != nil {
		return 0, err
	}

	s := Server{}
	gs := grpc.NewServer()
	RegisterFakeInitServer(gs, &s)

	err = gs.Serve(l)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func connect(addr string) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	sleep := 1 * time.Second
	for retry := 0; retry < 5; retry++ {
		conn, err = grpc.Dial(addr, grpc.WithInsecure(),
			grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				deadline, ok := ctx.Deadline()
				if ok {
					return net.DialTimeout("unix", addr, time.Until(deadline))
				}
				return net.Dial("unix", addr)
			}),
		)
		if err == nil {
			client := NewFakeInitClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			_, err = client.Ping(ctx, &PingRequest{})
			cancel()
			if err == nil {
				break
			}
		}
		time.Sleep(sleep)
		sleep = 2 * sleep
	}
	return conn, err
}

func cmdStart() (int, error) {
	addr := ""
	env := envVars{}

	endArg := len(os.Args)
	for i, v := range os.Args {
		if v == "--" {
			endArg = i
		}
	}
	fs := flag.NewFlagSet(fmt.Sprintf("%s start", os.Args[0]), flag.ExitOnError)
	fs.StringVar(&addr, "socket", "", "path to sock file to connect to")
	fs.Var(&env, "env", "--env K=V [--env=K=V...]")
	fs.Parse(os.Args[2:endArg])

	if addr == "" {
		return 0, fmt.Errorf("missing --socket argument")
	}

	conn, err := connect(addr)
	if err != nil {
		return 0, err
	}

	var args []string
	if endArg < len(os.Args) {
		args = os.Args[endArg+1:]
	}
	command := ""
	switch len(args) {
	case 0:
	case 1:
		command = args[0]
		args = nil
	default:
		command = args[0]
		args = args[1:]
	}

	client := NewFakeInitClient(conn)
	_, err = client.Start(context.Background(), &StartRequest{
		Command: command,
		Args:    args,
		Env:     []string(env),
	})
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func cmdExec() (exitCode int, errOut error) {
	addr := ""
	env := envVars{}

	endArg := len(os.Args)
	for i, v := range os.Args {
		if v == "--" {
			endArg = i
		}
	}
	fs := flag.NewFlagSet(fmt.Sprintf("%s exec", os.Args[0]), flag.ExitOnError)
	fs.StringVar(&addr, "socket", "", "path to sock file to connect to")
	fs.Var(&env, "env", "--env K=V [--env=K=V...]")
	fs.Parse(os.Args[2:endArg])

	if addr == "" {
		return 0, fmt.Errorf("missing --socket argument")
	}

	if term.Isatty(os.Stdin) {
		return 0, fmt.Errorf("TTY unsupported")
	}

	conn, err := connect(addr)
	if err != nil {
		return 0, err
	}

	var args []string
	if endArg < len(os.Args) {
		args = os.Args[endArg+1:]
	}
	command := ""
	switch len(args) {
	case 0:
	case 1:
		command = args[0]
		args = nil
	default:
		command = args[0]
		args = args[1:]
	}

	client := NewFakeInitClient(conn)
	stream, err := client.Exec(context.Background())
	if err != nil {
		return 0, err
	}
	defer stream.CloseSend()

	err = stream.Send(&ExecRequest{
		Type:    ExecRequestMessageType_EXEC_REQ_EXEC,
		Command: command,
		Args:    args,
		Env:     []string(env),
	})
	if err != nil {
		return 0, err
	}

	closeChan := make(chan struct{})
	go func() {
		buf := make([]byte, 512)
		for {
			select {
			case _, _ = <-closeChan:
				return
			default:
			}
			n, err := os.Stdin.Read(buf)
			if err == io.EOF {
				err = stream.Send(&ExecRequest{
					Type: ExecRequestMessageType_EXEC_REQ_STDIN,
				})
				if err != nil {
					log.Fatal(err)
				}
				return
			} else if err != nil {
				select {
				case _, _ = <-closeChan:
					return
				default:
				}
				log.Fatal(err)
			}
			if n == 0 {
				continue
			}
			err = stream.Send(&ExecRequest{
				Type:  ExecRequestMessageType_EXEC_REQ_STDIN,
				Stdin: buf[:n],
			})
			if err == io.EOF {
				return
			} else if err != nil {
				log.Fatal(err)
			}
		}
	}()

	go func() {
		sc := make(chan os.Signal, 1)
		defer func() {
			signal.Stop(sc)
			close(sc)
		}()
		signal.Notify(sc, syscall.SIGINT)

		for {
			select {
			case _, _ = <-closeChan:
				return
			case sig := <-sc:
				err = stream.Send(&ExecRequest{
					Type:   ExecRequestMessageType_EXEC_REQ_SIGNAL,
					Signal: int32(sig.(syscall.Signal)),
				})
				if err == io.EOF {
					return
				} else if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	defer close(closeChan)

	stderrClosed := false
	stdoutClosed := false
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return 0, io.ErrUnexpectedEOF
		} else if err != nil {
			return 0, err
		}
		switch res.Type {
		case ExecResponseMessageType_EXEC_RES_EXIT:
			return int(res.ExitCode), nil
		case ExecResponseMessageType_EXEC_RES_STDERR:
			if stderrClosed {
				continue
			}
			if len(res.Stderr) == 0 {
				os.Stderr.Close()
				stderrClosed = true
				continue
			}
			buf := bytes.NewBuffer(res.Stderr)
			io.Copy(os.Stderr, buf)
		case ExecResponseMessageType_EXEC_RES_STDOUT:
			if stdoutClosed {
				continue
			}
			if len(res.Stdout) == 0 {
				os.Stdout.Close()
				stdoutClosed = true
				continue
			}
			buf := bytes.NewBuffer(res.Stdout)
			io.Copy(os.Stdout, buf)
		}
	}

	return 0, nil
}

func cmdSignal() (int, error) {
	addr := ""
	sig := ""
	fs := flag.NewFlagSet(fmt.Sprintf("%s start", os.Args[0]), flag.ExitOnError)
	fs.StringVar(&addr, "socket", "", "path to sock file to connect to")
	fs.StringVar(&sig, "signal", "SIGINT", "signal to send to entrypoint")
	fs.Parse(os.Args[2:])

	if addr == "" {
		return 0, fmt.Errorf("missing --socket argument")
	}

	conn, err := connect(addr)
	if err != nil {
		return 0, err
	}

	s := syscall.SIGINT
	if i, err := strconv.Atoi(sig); err == nil {
		s = syscall.Signal(i)
	} else {
		original := sig
		sig = strings.ToUpper(sig)
		sig = strings.TrimPrefix(sig, "SIG")
		switch sig {
		case "ABRT":
			s = syscall.SIGABRT
		case "ALRM":
			s = syscall.SIGALRM
		case "BUS":
			s = syscall.SIGBUS
		case "CHLD":
			s = syscall.SIGCHLD
		case "CLD":
			s = syscall.SIGCLD
		case "CONT":
			s = syscall.SIGCONT
		case "FPE":
			s = syscall.SIGFPE
		case "HUP":
			s = syscall.SIGHUP
		case "ILL":
			s = syscall.SIGILL
		case "INT":
			s = syscall.SIGINT
		case "IO":
			s = syscall.SIGIO
		case "IOT":
			s = syscall.SIGIOT
		case "KILL":
			s = syscall.SIGKILL
		case "PIPE":
			s = syscall.SIGPIPE
		case "POLL":
			s = syscall.SIGPOLL
		case "PROF":
			s = syscall.SIGPROF
		case "PWR":
			s = syscall.SIGPWR
		case "QUIT":
			s = syscall.SIGQUIT
		case "SEGV":
			s = syscall.SIGSEGV
		case "STKFLT":
			s = syscall.SIGSTKFLT
		case "STOP":
			s = syscall.SIGSTOP
		case "SYS":
			s = syscall.SIGSYS
		case "TERM":
			s = syscall.SIGTERM
		case "TRAP":
			s = syscall.SIGTRAP
		case "TSTP":
			s = syscall.SIGTSTP
		case "TTIN":
			s = syscall.SIGTTIN
		case "TTOU":
			s = syscall.SIGTTOU
		case "UNUSED":
			s = syscall.SIGUNUSED
		case "URG":
			s = syscall.SIGURG
		case "USR1":
			s = syscall.SIGUSR1
		case "USR2":
			s = syscall.SIGUSR2
		case "VTALRM":
			s = syscall.SIGVTALRM
		case "WINCH":
			s = syscall.SIGWINCH
		case "XCPU":
			s = syscall.SIGXCPU
		case "XFSZ":
			s = syscall.SIGXFSZ
		default:
			return 0, fmt.Errorf("unknown signal %s", original)
		}
	}

	client := NewFakeInitClient(conn)
	_, err = client.Signal(context.Background(), &SignalRequest{
		Signal: int32(s),
	})
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func cmdStatus() (int, error) {
	addr := ""
	fs := flag.NewFlagSet(fmt.Sprintf("%s start", os.Args[0]), flag.ExitOnError)
	fs.StringVar(&addr, "socket", "", "path to sock file to connect to")
	fs.Parse(os.Args[2:])

	if addr == "" {
		return 0, fmt.Errorf("missing --socket argument")
	}

	conn, err := connect(addr)
	if err != nil {
		return 0, err
	}

	client := NewFakeInitClient(conn)
	s, err := client.Status(context.Background(), &StatusRequest{})
	if err != nil {
		return 0, err
	}

	out := struct {
		Status string `json:"status"`
		Pid    int    `json:"pid"`
	}{
		Pid: int(s.Pid),
	}
	switch s.Status {
	case Status_WAITING:
		out.Status = "waiting"
	case Status_RUNNING:
		out.Status = "running"
	}

	return 0, json.NewEncoder(os.Stdout).Encode(out)
}

type envVars []string

func (env *envVars) String() string {
	return "environment values"
}

func (env *envVars) Set(v string) error {
	if v == "" {
		return fmt.Errorf("env value cannot be empty must be in the form K=V where V is optional.")
	}
	*env = append(*env, v)
	return nil
}
