package server

import (
	"context"
	"net"
	"net/http"

	"github.com/puppetlabs/horsehead/netutil"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadata-api/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadata-api/data/metadata"
	"github.com/puppetlabs/nebula-tasks/pkg/metadata-api/data/secrets"
)

type Server struct {
	bindAddr string
	sec      *secrets.Store
	md       *metadata.Metadata
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// routing tree:
	// if the path is empty, then 404
	// otherwise, see if the next path segment matches "secret",
	// if it does, then route to the secret http sever to handle secret fetching
	// do the same if it matches "metadata", but instead route to the metadata http sever
}

func (s *Server) Run(ctx context.Context) errors.Error {
	ln, err := net.Listen("tcp", s.bindAddr)
	if err != nil {
		return errors.NewMetadataAPIServerError().WithCause(err)
	}

	ln = netutil.NewTCPKeepAliveListener(ln.(*net.TCPListener))
	ln = netutil.NewContextListener(ctx, ln)

	srv := &http.Server{Handler: s}

	defer srv.Shutdown(ctx)

	if err := srv.Serve(ln); err != nil {
		return errors.NewMetadataAPIServerError().WithCause(err)
	}

	return nil
}

func New(cfg *config.Config, sec *secrets.Store, md *metadata.Metadata) *Server {
	return &Server{
		bindAddr: cfg.BindAddr,
		sec:      sec,
		md:       md,
	}
}
