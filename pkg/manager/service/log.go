package service

import (
	"context"
	"time"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
)

type LogManager struct {
	logClient  plspb.LogClient
	logContext string
}

var _ model.LogManager = &LogManager{}

func (m *LogManager) PostLog(ctx context.Context, lcr *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	context := lcr.Context
	if context == "" {
		context = m.logContext
	}

	if m.logClient != nil {
		return m.logClient.Create(ctx, lcr)
	}

	return nil, nil
}

func (m *LogManager) PostLogMessage(ctx context.Context, message *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if m.logClient != nil {
		return m.logClient.MessageAppend(ctx, message)
	}

	return nil, nil
}

type LogManagerOption func(lm *LogManager)

func NewLogManager(logClient plspb.LogClient, logContext string, opts ...LogManagerOption) *LogManager {
	lm := &LogManager{
		logClient:  logClient,
		logContext: logContext,
	}

	for _, opt := range opts {
		opt(lm)
	}

	return lm
}
