package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"sort"
	"strings"

	"gopkg.in/gomail.v2"
)

const LocalName = "nebula.puppet.com"

type Sender struct {
	Client *smtp.Client
}

func (s *Sender) Send(from string, tos []string, wt io.WriterTo) (err error) {
	if err := s.Client.Mail(from); err != nil {
		return err
	}

	for _, to := range tos {
		if err := s.Client.Rcpt(to); err != nil {
			return err
		}
	}

	w, err := s.Client.Data()
	if err != nil {
		return err
	}
	defer func() {
		cerr := w.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err := wt.WriteTo(w); err != nil {
		return err
	}

	return nil
}

func (s *Sender) Close() error {
	return s.Client.Close()
}

type Dialer struct{}

func (Dialer) DialContext(ctx context.Context, spec ServerSpec) (sender gomail.SendCloser, err error) {
	port := spec.Port
	if port == 0 {
		port = 25 // SMTP
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", spec.Host, port))
	if err != nil {
		return nil, err
	}

	var useTLS bool
	if spec.TLS != nil {
		useTLS = *spec.TLS
	} else {
		useTLS = port == 465
	}

	tlsConfig := &tls.Config{
		ServerName: spec.Host,
	}

	if useTLS {
		conn = tls.Client(conn, tlsConfig)
	}

	c, err := smtp.NewClient(conn, spec.Host)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	if err := c.Hello(LocalName); err != nil {
		return nil, err
	}

	if !useTLS {
		// Then we must use STARTTLS.
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return nil, ErrSTARTTLSNotSupported
		}

		if err := c.StartTLS(tlsConfig); err != nil {
			return nil, err
		}
	}

	ok, auths := c.Extension("AUTH")
	if !ok {
		return nil, ErrAuthNotSupported
	}

	auth, err := chooseAuthMechanism(auths, spec)
	if err != nil {
		return nil, err
	}

	if err := c.Auth(auth); err != nil {
		return nil, err
	}

	return &Sender{Client: c}, nil
}

type loginAuth struct {
	spec ServerSpec
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.spec.Username), nil
		case "Password:":
			return []byte(a.spec.Password), nil
		}
	}

	return nil, nil
}

var authFactories = map[string]func(spec ServerSpec) smtp.Auth{
	"CRAM-MD5": func(spec ServerSpec) smtp.Auth {
		return smtp.CRAMMD5Auth(spec.Username, spec.Password)
	},
	"PLAIN": func(spec ServerSpec) smtp.Auth {
		return smtp.PlainAuth("", spec.Username, spec.Password, spec.Host)
	},
	"LOGIN": func(spec ServerSpec) smtp.Auth {
		return &loginAuth{spec}
	},
}

func chooseAuthMechanism(auths string, spec ServerSpec) (smtp.Auth, error) {
	al := sort.StringSlice(strings.Fields(auths))
	al.Sort()

	for _, pref := range []string{"CRAM-MD5", "PLAIN", "LOGIN"} {
		if r := al.Search(pref); r == len(al) || al[r] != pref {
			continue
		}

		return authFactories[pref](spec), nil
	}

	return nil, ErrNoAuthMechanismsAvailable
}
