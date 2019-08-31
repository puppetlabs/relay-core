package main

import (
	"encoding/json"
	"net/mail"
)

type AddressSpec struct {
	*mail.Address
}

func (as *AddressSpec) UnmarshalJSON(data []byte) (err error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	as.Address, err = mail.ParseAddress(s)
	return
}

type ServerSpec struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      *bool  `json:"tls"`
}

type BodySpec struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

type Spec struct {
	Server         ServerSpec    `json:"server"`
	From           AddressSpec   `json:"from"`
	To             []AddressSpec `json:"to"`
	Cc             []AddressSpec `json:"cc"`
	Bcc            []AddressSpec `json:"bcc"`
	Subject        string        `json:"subject"`
	Body           BodySpec      `json:"body"`
	TimeoutSeconds uint          `json:"timeoutSeconds"`
}
