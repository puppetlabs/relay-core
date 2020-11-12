package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
)

type logKey struct {
	context string
	name    string
}

type logMessages struct {
	id       string
	messages []string
}

type LogManager struct {
	mut sync.RWMutex

	ids  map[logKey]string
	logs map[string]logMessages
}

var _ model.LogManager = &LogManager{}

func (m *LogManager) PostLog(ctx context.Context, lcr *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	key := logKey{context: lcr.GetContext(), name: lcr.GetName()}
	id, found := m.ids[key]
	if found {
		return &plspb.LogCreateResponse{
			LogId: id,
		}, nil
	}

	id = uuid.New().String()

	m.ids[key] = id
	m.logs[id] = logMessages{
		id:       id,
		messages: make([]string, 0),
	}

	return &plspb.LogCreateResponse{
		LogId: id,
	}, nil
}

func (m *LogManager) PostLogMessage(ctx context.Context, message *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if message == nil {
		return nil, nil
	}

	logs, found := m.logs[message.GetLogId()]
	if !found {
		return nil, model.ErrNotFound
	}

	logs.messages = append(logs.messages, string(message.Payload))
	m.logs[message.GetLogId()] = logs

	return &plspb.LogMessageAppendResponse{
		LogId:        logs.id,
		LogMessageId: uuid.New().String(),
	}, nil
}

type LogManagerOption func(lm *LogManager)

func NewLogManager(opts ...LogManagerOption) *LogManager {
	lm := &LogManager{
		ids:  make(map[logKey]string),
		logs: make(map[string]logMessages),
	}

	for _, opt := range opts {
		opt(lm)
	}

	return lm
}
