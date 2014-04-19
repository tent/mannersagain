// Package mannersagain combines manners and goagain to provide graceful hot
// restarting of net/http servers.
package mannersagain

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/braintree/manners"
	"github.com/rcrowley/goagain"
)

func newListener(l net.Listener) net.Listener {
	return listener{Listener: l, closed: make(chan struct{})}
}

type listener struct {
	net.Listener
	closed chan struct{}
}

var ErrClosed = errors.New("mannersagain: listener has been gracefully closed")

func (l listener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.closed:
			return nil, ErrClosed
		default:
		}

		// Set a deadline so Accept doesn't block forever, which gives
		// us an opportunity to stop gracefully.
		l.Listener.(*net.TCPListener).SetDeadline(time.Now().Add(100e6))
		c, err := l.Listener.Accept()
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			continue
		}
		return c, err
	}
}

func (l listener) Close() error {
	close(l.closed)
	return nil
}

func ListenAndServe(addr string, handler http.Handler) error {
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
		gl = manners.NewListener(newListener(l), srv)
		go serve(gl)
	} else {
		log.Println("Resuming listening on", l.Addr())
		gl = manners.NewListener(newListener(l), srv)
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

	if goagain.Strategy == goagain.Double {
		// If we received SIGUSR2, re-exec the parent process.
		if sig == goagain.SIGUSR2 {
			if err := goagain.Exec(l); err != nil {
				return err
			}
		}
	}

	return nil
}
