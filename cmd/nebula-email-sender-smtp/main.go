package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"gopkg.in/gomail.v2"
)

func main() {
	specURL := flag.String("spec-url", os.Getenv(taskutil.SpecURLEnvName), "url to fetch the spec from")

	flag.Parse()

	planOpts := taskutil.DefaultPlanOptions{SpecURL: *specURL}

	var spec Spec
	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, planOpts); err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()
	if spec.TimeoutSeconds != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, time.Duration(spec.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	m := gomail.NewMessage(gomail.SetCharset("UTF-8"))
	m.SetHeader("From", spec.From.String())

	if spec.Subject != "" {
		m.SetHeader("Subject", spec.Subject)
	}

	tos := []struct {
		Field        string
		AddressSpecs []AddressSpec
	}{
		{Field: "To", AddressSpecs: spec.To},
		{Field: "Cc", AddressSpecs: spec.Cc},
		{Field: "Bcc", AddressSpecs: spec.Bcc},
	}
	for _, to := range tos {
		s := make([]string, len(to.AddressSpecs))
		for i, spec := range to.AddressSpecs {
			s[i] = spec.String()
		}

		m.SetHeader(to.Field, s...)
	}

	alternatives := []struct {
		MediaType string
		Data      string
	}{
		{MediaType: "text/plain", Data: spec.Body.Text},
		{MediaType: "text/html", Data: spec.Body.HTML},
	}
	for _, alt := range alternatives {
		if alt.Data == "" {
			continue
		}

		m.AddAlternative(alt.MediaType, alt.Data)
	}

	var dialer Dialer
	sender, err := dialer.DialContext(ctx, spec.Server)
	if err != nil {
		log.Fatalln("could not connect to SMTP server:", err)
	}

	if err := gomail.Send(sender, m); err != nil {
		log.Fatalln("could not send message:", err)
	}
}
