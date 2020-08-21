package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"

	"github.com/gonuts/go-shellquote"
)

func main() {
	l, err := net.Listen("tcp", ":4321")
	if err != nil {
		log.Fatal(err)
	}

	mut := sync.Mutex{}
	cmd := ""
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		mut.Lock()
		defer mut.Unlock()

		if cmd != "" {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		err := l.Close()
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		cmd = string(body)
		w.WriteHeader(http.StatusOK)
	})

	s := &http.Server{}
	err = s.Serve(l)
	mut.Lock()
	if err != nil && cmd == "" {
		log.Fatal(err)
	}

	args, err := shellquote.Split(cmd)
	if err != nil {
		log.Fatal(err)
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Start()
	if exit, ok := err.(*exec.ExitError); ok {
		os.Exit(exit.ExitCode())
	} else if err != nil {
		log.Fatal(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc)
	go func() {
		for {
			sig, ok := <-sc
			if !ok {
				return
			}
			c.Process.Signal(sig)
		}
	}()

	err = c.Wait()
	if exit, ok := err.(*exec.ExitError); ok {
		os.Exit(exit.ExitCode())
	} else if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
