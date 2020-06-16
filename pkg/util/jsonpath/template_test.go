package jsonpath_test

// Portions of this file are derived from the JSONPath implementation in
// Kubernetes client-go.
//
// Copyright 2015 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/util/jsonpath"
	"github.com/stretchr/testify/require"
)

type test struct {
	Name       string
	Data       interface{}
	Expr       string
	NeedsSort  bool
	Expected   string
	ParseError string
	EvalError  string
}

func (tt *test) Run(t *testing.T) {
	eval, err := jsonpath.TemplateLanguage().NewEvaluable(tt.Expr)
	if tt.ParseError != "" {
		require.NotNil(t, err)
		require.Contains(t, err.Error(), tt.ParseError)
		return
	} else {
		require.NoError(t, err)
	}

	r, err := eval.EvalString(context.Background(), tt.Data)

	if tt.NeedsSort {
		rs := strings.Fields(r)
		sort.Strings(rs)

		expected := strings.Fields(tt.Expected)
		sort.Strings(expected)

		require.Equal(t, expected, rs)
	} else {
		require.Equal(t, tt.Expected, r)
	}

	if tt.EvalError != "" {
		require.NotNil(t, err)
		require.Contains(t, err.Error(), tt.EvalError)
	} else {
		require.NoError(t, err)
	}
}

type tests []test

func (tts tests) RunAll(t *testing.T) {
	for _, tt := range tts {
		t.Run(tt.Name, tt.Run)
	}
}

func TestBasic(t *testing.T) {
	tests{
		{
			Name:     "literal",
			Data:     nil,
			Expr:     "this\nis  some ~text~\n",
			Expected: "this\nis  some ~text~\n",
		},
		{
			Name:     "embedded using $",
			Data:     map[string]interface{}{"foo": "world"},
			Expr:     "hello { $.foo }",
			Expected: "hello world",
		},
		{
			Name:     "range",
			Data:     map[string]interface{}{"foo": []interface{}{"bar", "baz", "quux"}},
			Expr:     `start{range .foo} -> {$} <-{end} end`,
			Expected: "start -> bar <- -> baz <- -> quux <- end",
		},
		{
			Name: "nested range",
			Data: map[string]interface{}{
				"foo": []interface{}{
					[]interface{}{"a", "b"},
					[]interface{}{"c", "d"},
				},
			},
			Expr:     `{range .foo}{range @}{@}{end}{end}`,
			Expected: "abcd",
		},
		{
			Name:       "too few range ends",
			Expr:       `{range .foo}{$}`,
			ParseError: "unexpected EOF while scanning JSONPath template range end",
		},
		{
			Name:       "too many range ends",
			Expr:       `{range .foo}{$}{end}{end}`,
			ParseError: "unexpected Ident while scanning operator",
		},
		{
			Name:     "single quote",
			Expr:     `{'{}'}`,
			Expected: "{}",
		},
	}.RunAll(t)
}

func TestJSONObject(t *testing.T) {
	var storeData interface{}
	err := json.Unmarshal([]byte(`{
		"Name": "jsonpath",
		"Book": [
			{"Category": "reference", "Author": "Nigel Rees", "Title": "Sayings of the Century", "Price": 8.95},
			{"Category": "fiction", "Author": "Evelyn Waugh", "Title": "Sword of Honour", "Price": 12.99},
			{"Category": "fiction", "Author": "Herman Melville", "Title": "Moby Dick", "Price": 8.99}
		],
		"Bicycle": [
			{"Color": "red", "Price": 19.95, "IsNew": true},
			{"Color": "green", "Price": 20.01, "IsNew": false}
		],
		"Labels": {
			"engineer": 10,
			"web/html": 15,
			"k8s-app": 20
		},
		"Employees": {
			"jason": "manager",
			"dan": "clerk"
		}
	}`), &storeData)
	require.NoError(t, err)

	tests{
		{
			Name:     "plain",
			Data:     nil,
			Expr:     "hello jsonpath",
			Expected: "hello jsonpath",
		},
		{
			Name:     "recursive",
			Data:     []interface{}{1, 2, 3},
			Expr:     "{..}",
			Expected: "[1 2 3] 1 2 3",
		},
		{
			Name:     "filter",
			Data:     []interface{}{2, 6, 3, 7},
			Expr:     "{[?(@<5)]}",
			Expected: "2 3",
		},
		{
			Name:     "quote",
			Data:     nil,
			Expr:     `{"{"}`,
			Expected: "{",
		},
		{
			Name:     "union",
			Data:     []interface{}{0, 1, 2, 3, 4},
			Expr:     "{[1,3,4]}",
			Expected: "1 3 4",
		},
		{
			Name:     "array",
			Data:     []interface{}{"Monday", "Tuesday"},
			Expr:     "{[0:2]}",
			Expected: "Monday Tuesday",
		},
		{
			Name:     "variable",
			Data:     storeData,
			Expr:     "hello {.Name}",
			Expected: "hello jsonpath",
		},
		{
			Name:     "dict with slash",
			Data:     storeData,
			Expr:     "{$.Labels.web/html}",
			Expected: "15",
		},
		{
			Name:     "nested dict (1)",
			Data:     storeData,
			Expr:     "{$.Employees.jason}",
			Expected: "manager",
		},
		{
			Name:     "nested dict (2)",
			Data:     storeData,
			Expr:     "{$.Employees.dan}",
			Expected: "clerk",
		},
		{
			Name:     "dict with dash",
			Data:     storeData,
			Expr:     "{$.Labels.k8s-app}",
			Expected: "20",
		},
		{
			Name:     "nested",
			Data:     storeData,
			Expr:     "{.Bicycle[*].Color}",
			Expected: "red green",
		},
		{
			Name:     "concatenated array",
			Data:     storeData,
			Expr:     "{.Book[*].Author}",
			Expected: "Nigel Rees Evelyn Waugh Herman Melville",
		},
		{
			Name:     "all fields",
			Data:     storeData,
			Expr:     "{.Bicycle.*.Color}",
			Expected: "red green",
		},
		{
			Name:      "recursive fields",
			Data:      storeData,
			Expr:      "{..Price}",
			NeedsSort: true,
			Expected:  "8.95 12.99 8.99 19.95 20.01",
		},
		{
			Name:     "last element of array",
			Data:     storeData,
			Expr:     "{.Book[-1:].Author}",
			Expected: "Herman Melville",
		},
		{
			Name:     "recursive array",
			Data:     storeData,
			Expr:     "{..Book[2].Author}",
			Expected: "Herman Melville",
		},
		{
			Name:     "filter",
			Data:     storeData,
			Expr:     "{.Bicycle[?(@.IsNew==true)].Color}",
			Expected: "red",
		},
		{
			Name:     "nonexistent field",
			Data:     storeData,
			Expr:     "{.hello}",
			Expected: "",
		},
	}.RunAll(t)
}

func TestJSONArray(t *testing.T) {
	var points interface{}
	err := json.Unmarshal([]byte(`[
		{"id": "i1", "x":4, "y":-5},
		{"id": "i2", "x":-2, "y":-5, "z":1},
		{"id": "i3", "x":  8, "y":  3 },
		{"id": "i4", "x": -6, "y": -1 },
		{"id": "i5", "x":  0, "y":  2, "z": 1 },
		{"id": "i6", "x":  1, "y":  4 }
	]`), &points)
	require.NoError(t, err)

	tests{
		{
			Name:     "existence filter",
			Data:     points,
			Expr:     `{[?(@.z)].id}`,
			Expected: "i2 i5",
		},
		{
			Name:     "bracket key",
			Data:     points,
			Expr:     `{[0]['id']}`,
			Expected: "i1",
		},
	}.RunAll(t)
}

func TestKubernetes(t *testing.T) {
	var nodes interface{}
	err := json.Unmarshal([]byte(`{
		"kind": "List",
		"items":[
		  {
			"kind":"None",
			"metadata":{
			  "name":"127.0.0.1",
			  "labels":{
				"kubernetes.io/hostname":"127.0.0.1"
			  }
			},
			"status":{
			  "capacity":{"cpu":"4"},
			  "ready": true,
			  "addresses":[{"type": "LegacyHostIP", "address":"127.0.0.1"}]
			}
		  },
		  {
			"kind":"None",
			"metadata":{
			  "name":"127.0.0.2",
			  "labels":{
				"kubernetes.io/hostname":"127.0.0.2"
			  }
			},
			"status":{
			  "capacity":{"cpu":"8"},
			  "ready": false,
			  "addresses":[
				{"type": "LegacyHostIP", "address":"127.0.0.2"},
				{"type": "another", "address":"127.0.0.3"}
			  ]
			}
		  }
		],
		"users":[
		  {
			"name": "myself",
			"user": {}
		  },
		  {
			"name": "e2e",
			"user": {"username": "admin", "password": "secret"}
			}
		]
	}`), &nodes)
	require.NoError(t, err)

	tests{
		{
			Name:     "range item",
			Data:     nodes,
			Expr:     `{range .items[*]}{.metadata.name}, {end}{.kind}`,
			Expected: "127.0.0.1, 127.0.0.2, List",
		},
		{
			Name:     "range item with quote",
			Data:     nodes,
			Expr:     `{range .items[*]}{.metadata.name}{"\t"}{end}`,
			Expected: "127.0.0.1\t127.0.0.2\t",
		},
		{
			Name:     "range address",
			Data:     nodes,
			Expr:     `{.items[*].status.addresses[*].address}`,
			Expected: "127.0.0.1 127.0.0.2 127.0.0.3",
		},
		{
			Name:     "double_range",
			Data:     nodes,
			Expr:     `{range .items[*]}{range .status.addresses[*]}{.address}, {end}{end}`,
			Expected: "127.0.0.1, 127.0.0.2, 127.0.0.3, ",
		},
		{
			Name:     "item name",
			Data:     nodes,
			Expr:     `{.items[*].metadata.name}`,
			Expected: "127.0.0.1 127.0.0.2",
		},
		{
			Name:     "union nodes capacity",
			Data:     nodes,
			Expr:     `{.items[*]['metadata', 'status']['name', 'capacity']}`,
			Expected: "127.0.0.1 map[cpu:4] 127.0.0.2 map[cpu:8]",
		},
		{
			Name:     "range nodes capacity",
			Data:     nodes,
			Expr:     `{range .items[*]}[{.metadata.name}, {.status.capacity}] {end}`,
			Expected: "[127.0.0.1, map[cpu:4]] [127.0.0.2, map[cpu:8]] ",
		},
		{
			Name:     "user password",
			Data:     nodes,
			Expr:     `{.users[?(@.name=='e2e')].user.password}`,
			Expected: "secret",
		},
		{
			Name:     "hostname",
			Data:     nodes,
			Expr:     `{.items[0].metadata.labels['kubernetes.io/hostname']}`,
			Expected: "127.0.0.1",
		},
		{
			Name:     "hostname filter",
			Data:     nodes,
			Expr:     `{.items[?(@.metadata.labels['kubernetes.io/hostname']=="127.0.0.1")].kind}`,
			Expected: "None",
		},
		{
			Name:     "bool item",
			Data:     nodes,
			Expr:     `{.items[?(@..ready==true)].metadata.name}`,
			Expected: "127.0.0.1",
		},
		{
			Name:      "recursive name",
			Data:      nodes,
			Expr:      "{..name}",
			NeedsSort: true,
			Expected:  "127.0.0.1 127.0.0.2 myself e2e",
		},
	}.RunAll(t)
}
