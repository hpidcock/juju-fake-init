package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/gonuts/go-shellquote"
)

func main() {
	code, err := run()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

func run() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:4321")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	sshl, err := net.Listen("tcp", "127.0.0.1:4322")
	if err != nil {
		return 0, err
	}
	defer sshl.Close()
	cancel := make(chan struct{})
	defer close(cancel)

	ssh.Handle(func(s ssh.Session) {
		req, windowChannel, isPty := s.Pty()
		if !isPty {
			cmd := s.Command()
			if len(cmd) == 0 {
				io.WriteString(s, "No PTY requested.\n")
				s.Exit(1)
			}
			sc := make(chan ssh.Signal, 1)
			defer close(sc)
			s.Signals(sc)
			c := exec.Command(cmd[0], cmd[1:]...)
			c.Stdin = s
			c.Stdout = s
			c.Stderr = s.Stderr()
			err = c.Start()
			if exit, ok := err.(*exec.ExitError); ok {
				s.Exit(exit.ExitCode())
			} else if err != nil {
				log.Fatal(err)
			}
			go func() {
				for {
					sig, ok := <-sc
					if !ok {
						return
					}
					switch sig {
					case ssh.SIGABRT:
						c.Process.Signal(syscall.SIGABRT)
					case ssh.SIGALRM:
						c.Process.Signal(syscall.SIGALRM)
					case ssh.SIGFPE:
						c.Process.Signal(syscall.SIGFPE)
					case ssh.SIGHUP:
						c.Process.Signal(syscall.SIGHUP)
					case ssh.SIGILL:
						c.Process.Signal(syscall.SIGILL)
					case ssh.SIGINT:
						c.Process.Signal(syscall.SIGINT)
					case ssh.SIGKILL:
						c.Process.Signal(syscall.SIGKILL)
					case ssh.SIGPIPE:
						c.Process.Signal(syscall.SIGPIPE)
					case ssh.SIGQUIT:
						c.Process.Signal(syscall.SIGQUIT)
					case ssh.SIGSEGV:
						c.Process.Signal(syscall.SIGSEGV)
					case ssh.SIGTERM:
						c.Process.Signal(syscall.SIGTERM)
					case ssh.SIGUSR1:
						c.Process.Signal(syscall.SIGUSR1)
					case ssh.SIGUSR2:
						c.Process.Signal(syscall.SIGUSR2)
					}
				}
			}()
			err = c.Wait()
			if exit, ok := err.(*exec.ExitError); ok {
				s.Exit(exit.ExitCode())
			} else if err != nil {
				log.Fatal(err)
			}
			return
		}
		cmd := exec.Command("sh")
		cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", req.Term))
		f, err := pty.Start(cmd)
		if err != nil {
			io.WriteString(s, err.Error())
			s.Exit(1)
		}
		go func() {
			for win := range windowChannel {
				syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
					uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(win.Height), uint16(win.Width), 0, 0})))
			}
		}()
		go func() {
			io.Copy(f, s) // stdin
		}()
		io.Copy(s, f) // stdout
		cmd.Wait()
	})

	go func() {
		err = ssh.Serve(sshl, nil)
		select {
		case _, _ = <-cancel:
			return
		default:
		}
		if err != nil {
			log.Fatal(err)
		}
	}()

	ready := make(chan struct{})
	cmd := ""
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		select {
		case _, _ = <-ready:
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		default:
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		cmd = string(body)
		close(ready)
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		s := &http.Server{}
		err := s.Serve(l)
		select {
		case _, _ = <-cancel:
			return
		default:
		}
		if err != nil {
			log.Fatal(err)
		}
	}()

	_, _ = <-ready

	args, err := shellquote.Split(cmd)
	if err != nil {
		return 0, err
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Start()
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), nil
	} else if err != nil {
		return 0, err
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc)
	go func() {
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

	err = c.Wait()
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), nil
	} else if err != nil {
		return 0, err
	}
	return 0, nil
}
