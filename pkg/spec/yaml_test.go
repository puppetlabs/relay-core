package spec_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/stretchr/testify/require"
)

type yamlTest struct {
	Name          string
	Data          string
	ExpectedTree  any
	ExpectedError error
}

func (tt yamlTest) Run(t *testing.T) {
	tree, err := spec.ParseYAMLString(tt.Data)
	if tt.ExpectedError != nil {
		require.Equal(t, tt.ExpectedError, err)
	} else {
		require.NoError(t, err)
	}
	require.Equal(t, tt.ExpectedTree, tree)
}

type yamlTests []yamlTest

func (tts yamlTests) RunAll(t *testing.T) {
	for _, tt := range tts {
		t.Run(tt.Name, tt.Run)
	}
}

var yamlCleanPattern = regexp.MustCompile(`\n(\t*)[^\t]`)

func yaml(in string) string {
	// Find leading indent.
	strip := -1

	matches := yamlCleanPattern.FindAllStringSubmatch(in, -1)
	for _, match := range matches {
		if strip < 0 || len(match[1]) < strip {
			strip = len(match[1])
		}
	}

	// Remove leading indent.
	if strip >= 0 {
		in = yamlCleanPattern.ReplaceAllStringFunc(in, func(match string) string {
			return "\n" + match[1+strip:]
		})
	}

	// Replace tabs with spaces (tabs are not valid YAML).
	in = strings.ReplaceAll(in, "\t", "  ")

	// Trim trailing whitespace.
	return strings.TrimRightFunc(in, func(r rune) bool { return unicode.IsSpace(r) })
}

func TestYAML(t *testing.T) {
	yamlTests{
		{
			Name: "basic",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": "something",
			},
		},
		{
			Name: "secret scalar",
			Data: yaml(`
				aws:
					accessKeyID: foo
					secretAccessKey: !Secret secretAccessKey
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "foo",
					"secretAccessKey": jsonSecret("secretAccessKey"),
				},
				"op": "something",
			},
		},
		{
			Name: "secret sequence",
			Data: yaml(`
				aws:
					accessKeyID: foo
					secretAccessKey: !Secret [secretAccessKey]
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "foo",
					"secretAccessKey": jsonSecret("secretAccessKey"),
				},
				"op": "something",
			},
		},
		{
			Name: "secret mapping",
			Data: yaml(`
				aws:
					accessKeyID: foo
					secretAccessKey: !Secret {name: secretAccessKey}
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "foo",
					"secretAccessKey": jsonSecret("secretAccessKey"),
				},
				"op": "something",
			},
		},
		{
			Name: "connection sequence",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Connection [aws, test-creds]
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": jsonConnection("aws", "test-creds"),
			},
		},
		{
			Name: "connection mapping",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Connection {type: aws, name: test-creds}
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": jsonConnection("aws", "test-creds"),
			},
		},
		{
			Name: "output sequence",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Output [prev, operation]
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": jsonOutput("prev", "operation"),
			},
		},
		{
			Name: "output mapping",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Output {from: prev, name: operation}
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": jsonOutput("prev", "operation"),
			},
		},
		{
			Name: "parameter scalar",
			Data: yaml(`
				aws:
					accessKeyID: !Parameter accessKeyID
					region: us-west-2
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": jsonParameter("accessKeyID"),
					"region":      "us-west-2",
				},
				"op": "something",
			},
		},
		{
			Name: "parameter sequence",
			Data: yaml(`
				aws:
					accessKeyID: !Parameter [accessKeyID]
					region: us-west-2
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": jsonParameter("accessKeyID"),
					"region":      "us-west-2",
				},
				"op": "something",
			},
		},
		{
			Name: "parameter mapping",
			Data: yaml(`
				aws:
					accessKeyID: !Parameter {name: accessKeyID}
					region: us-west-2
				op: something
			`),
			ExpectedTree: map[string]any{
				"aws": map[string]any{
					"accessKeyID": jsonParameter("accessKeyID"),
					"region":      "us-west-2",
				},
				"op": "something",
			},
		},
		{
			Name: "conditional invocation",
			Data: yaml(`
				- !Fn.equals [!Parameter param1, "foobar"]
				- !Fn.notEquals [!Parameter param2, "barfoo"]
				`),
			ExpectedTree: []any{
				jsonInvocation("equals", []any{
					jsonParameter("param1"), "foobar",
				}),
				jsonInvocation("notEquals", []any{
					jsonParameter("param2"), "barfoo",
				}),
			},
		},
		{
			Name: "invocation scalar",
			Data: yaml(`
				message: !Fn.jsonUnmarshal '{"foo": "bar"}'
			`),
			ExpectedTree: map[string]any{
				"message": jsonInvocation("jsonUnmarshal", []any{`{"foo": "bar"}`}),
			},
		},
		{
			Name: "invocation sequence",
			Data: yaml(`
				message: !Fn.concat ["Hello, ", !Parameter name]
			`),
			ExpectedTree: map[string]any{
				"message": jsonInvocation("concat", []any{
					"Hello, ",
					jsonParameter("name"),
				}),
			},
		},
		{
			Name: "invocation mapping",
			Data: yaml(`
				message: !Fn.getMessage {host: api.example.com}
			`),
			ExpectedTree: map[string]any{
				"message": jsonInvocation("getMessage", map[string]any{"host": "api.example.com"}),
			},
		},
		{
			Name: "binary encoding",
			Data: yaml(`
				message: !!binary SGVsbG8sIJCiikU=
			`),
			ExpectedTree: map[string]any{
				"message": jsonEncoding("base64", "SGVsbG8sIJCiikU="),
			},
		},
		{
			Name: "invalid tag",
			Data: yaml(`
				message: !Fetch ball
			`),
			ExpectedError: fmt.Errorf(`unknown tag "!Fetch"`),
		},
	}.RunAll(t)
}
