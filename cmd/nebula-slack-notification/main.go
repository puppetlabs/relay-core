package main

import (
	"fmt"
	"github.com/puppetlabs/nebula-tasks/pkg/notify/slack"
	"os"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"errors"
)

type Spec struct {
	Apitoken string `json:"apitoken"`
	Channel  string `json:"channel"`
	Message  string `json:"message"`
	Username string `json:"username"`
}

func getSpec() (*Spec, error) {
	specUrl := os.Getenv("SPEC_URL")
	if "" == specUrl {
		return nil, errors.New("Missing required environment variable: SPEC_URL")
	}
	resp, err := http.Get(specUrl)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("GET %s: %v", specUrl, err))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("GET %s: %v", specUrl, err))
	}
	if resp.StatusCode / 100 != 2 {
		return nil, errors.New(fmt.Sprintf("GET %s -> %d: %v", specUrl, resp.StatusCode, err))
	}
	var spec Spec
	err = json.Unmarshal([]byte(body), &spec)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("GET %s -> %s: %v", specUrl, body, err))
	}
	return &spec, nil
}

func main() {
	spec, err := getSpec()
	if nil != err {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	if "" == spec.Channel || "" == spec.Message || "" == spec.Apitoken {
		fmt.Printf("Missing required fields. Expect spec to contain 'apitoken', 'channel' and 'message', got %v\n", *spec)
		os.Exit(1)
	}
	if "" == spec.Username {
		spec.Username = "Nebula"
	}

	client := slack.New(spec.Apitoken)
	res, err := client.Chat().PostMessage(
		slack.PostMessageRequest{
			Channel:  spec.Channel,
			Text:     spec.Message,
			Username: spec.Username,
		})

	if nil != err {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Response: %+v\n", res)
	os.Exit(0)
}
