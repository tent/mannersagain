// Package mannersagain combines manners and goagain to provide graceful hot
// restarting of net/http servers.
package mannersagain

import (
	"log"
	"net"
	"net/http"

	"github.com/braintree/manners"
	"github.com/rcrowley/goagain"
)

type nopCloser struct {
	net.Listener
}

func (nopCloser) Close() error { return nil }

func ListenAndServe(addr string, handler http.Handler) error {
	goagain.Strategy = goagain.Double
	var gl *manners.GracefulListener
	srv := manners.NewServer()

	done := make(chan struct{})
	serve := func(l net.Listener) {
		srv.Serve(l, handler)
		close(done)
	}

	// Attempt to inherit a listener from our parent
	l, err := goagain.Listener()
	if err != nil {
		// We don't have an inherited listener, create a new one
		l, err = net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Println("Listening on", l.Addr())
		gl = manners.NewListener(nopCloser{l}, srv)
		go serve(gl)
	} else {
		log.Println("Resuming listening on", l.Addr())
		gl = manners.NewListener(nopCloser{l}, srv)
		go serve(gl)

		// If this is the child, send the parent SIGUSR2. If this is the
		// parent, send the child SIGQUIT.
		if err := goagain.Kill(); nil != err {
			return err
		}
	}

	// Block the main goroutine awaiting signals.
	sig, err := goagain.Wait(l)
	if err != nil {
		return err
	}

	// Stop accepting new connections
	gl.Close()
	// Wait for all existing connections to complete
	<-done

	// If we received SIGUSR2, re-exec the parent process.
	if sig == goagain.SIGUSR2 {
		if err := goagain.Exec(l); err != nil {
			return err
		}
	}

	return nil
}
