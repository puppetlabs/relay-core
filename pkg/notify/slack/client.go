package slack

import (
	log "github.com/inconshreveable/log15"
	"net/http"
)

type Client struct {
	token    string
	endpoint string
	log      log.Logger
	client   *http.Client
}

type Option func(*Client)

func OptionAPIURL(url string) Option {
	return func(client *Client) {
		client.endpoint = url
	}
}

func OptionLogger(logger log.Logger) Option {
	return func(client *Client) {
		client.log = logger
	}
}

func OptionHttpClient(httpClient *http.Client) Option {
	return func(client *Client) {
		client.client = httpClient
	}
}

func New(token string, options ...Option) *Client {
	client := &Client{
		token:    token,
		endpoint: "https://slack.com/api/",
		client:   &http.Client{},
		log:      log.New("module", "notify/slack"),
	}
	for _, opt := range options {
		opt(client)
	}
	return client
}

type Chat Client

func (c *Client) Chat() *Chat {
	chat := Chat(*c)
	return &chat
}
