package slack

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
)

type PostMessageRequest struct {
	Channel  string
	Text     string
	Username string
}
type PostMessageResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

func (c *Chat) PostMessage(req PostMessageRequest) (*PostMessageResponse, error) {
	post := url.Values{}
	if len(req.Channel) > 0 {
		post.Set("channel", req.Channel)
	}
	if len(req.Text) > 0 {
		post.Set("text", req.Text)
	}
	if len(req.Username) > 0 {
		post.Set("username", req.Username)
	}
	post.Set("token", c.token)
	res, err := c.client.PostForm(c.endpoint+"chat.postMessage", post)
	if nil != err {
		return nil, err
	}
	defer res.Body.Close()
	bytes, err := ioutil.ReadAll(res.Body)
	if nil == bytes {
		return nil, err
	}
	if res.StatusCode/100 != 2 {
		return nil, errors.New("Unexpected status code in chat.postMessage: " + string(bytes))
	}

	c.log.Debug("chat.postMessage", "req", req, "res", string(bytes))

	var postMessageResponse PostMessageResponse
	if err := json.Unmarshal(bytes, &postMessageResponse); nil != err {
		return nil, err
	}

	if !postMessageResponse.Ok {
		return nil, errors.New(postMessageResponse.Error)
	}

	return &postMessageResponse, nil
}
