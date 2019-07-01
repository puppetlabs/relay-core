package main

import (
	"fmt"
	"flag"
	"github.com/puppetlabs/nebula-tasks/pkg/notify/slack"
	"os"
)

var channel = flag.String("channel", "", "The slack channel to notify")
var message = flag.String("message", "", "The message to notify the channel with")
var username = flag.String("username", "Nebula", "The username to publish the message as")

func main() {
	flag.Parse()
	if "" == *channel || "" == *message {
		fmt.Println("Missing required flag(s)")
		flag.PrintDefaults()
		os.Exit(1)
	}
	token := os.Getenv("SLACK_TOKEN")
	if "" == token {
		fmt.Println("Missing required environment variable: SLACK_TOKEN")
		os.Exit(1)
	}
	client := slack.New(token)
	res, err := client.Chat().PostMessage(
		slack.PostMessageRequest{
			Channel:  *channel,
			Text:     *message,
			Username: *username,
		})

	if nil != err {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Printf("Response: %+v\n", res)
	os.Exit(0)
}
