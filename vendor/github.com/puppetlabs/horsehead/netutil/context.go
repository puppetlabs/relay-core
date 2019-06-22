package netutil

import (
	"context"
	"net"
	"sync"
)

// ContextListener is a net.Listener that closes itself when the context it was
// created with concludes.
type ContextListener struct {
	net.Listener

	closeMut sync.Mutex
	closeErr *error
	closeCh  chan struct{}
}

func (ln *ContextListener) Close() error {
	ln.closeMut.Lock()
	defer ln.closeMut.Unlock()

	select {
	case <-ln.closeCh:
		if ln.closeErr != nil {
			err := *ln.closeErr
			ln.closeErr = nil
			return err
		}
	default:
		close(ln.closeCh)
	}

	return ln.Listener.Close()
}

func (ln *ContextListener) waitContext(ctx context.Context) {
	select {
	case <-ln.closeCh:
		return
	case <-ctx.Done():
	}

	ln.closeMut.Lock()
	defer ln.closeMut.Unlock()

	select {
	case <-ln.closeCh:
		return
	default:
		close(ln.closeCh)
	}

	err := ln.Listener.Close()
	ln.closeErr = &err
}

func NewContextListener(ctx context.Context, delegate net.Listener) net.Listener {
	ln := &ContextListener{
		Listener: delegate,

		closeCh: make(chan struct{}),
	}
	go ln.waitContext(ctx)

	return ln
}
