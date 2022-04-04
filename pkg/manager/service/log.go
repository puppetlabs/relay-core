package service

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
	"google.golang.org/protobuf/proto"
)

type LogManager struct {
	logClient  plspb.LogClient
	logContext string
}

var _ model.LogManager = &LogManager{}

func (m *LogManager) PostLog(ctx context.Context, value interface{}) ([]byte, error) {
	incoming, ok := value.(string)
	if ok {
		request := &plspb.LogCreateRequest{}

		err := proto.Unmarshal([]byte(incoming), request)
		if err != nil {
			return nil, err
		}

		if request.Context == "" {
			request.Context = m.logContext
		}

		if m.logClient != nil {
			response, err := m.logClient.Create(ctx, request)
			if err != nil {
				return nil, err
			}

			return proto.Marshal(response)
		}
	}

	return nil, nil
}

func (m *LogManager) PostLogMessage(ctx context.Context, logID string, value interface{}) ([]byte, error) {
	incoming, ok := value.(string)
	if ok {
		request := &plspb.LogMessageAppendRequest{}

		err := proto.Unmarshal([]byte(incoming), request)
		if err != nil {
			return nil, err
		}

		if m.logClient != nil {
			response, err := m.logClient.MessageAppend(ctx, request)
			if err != nil {
				return nil, err
			}

			return proto.Marshal(response)
		}
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
