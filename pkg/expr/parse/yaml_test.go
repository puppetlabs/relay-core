package parse_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/puppetlabs/nebula-tasks/pkg/expr/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/testutil"
	"github.com/stretchr/testify/require"
)

type yamlTest struct {
	Name          string
	Data          string
	ExpectedTree  parse.Tree
	ExpectedError error
}

func (tt yamlTest) Run(t *testing.T) {
	tree, err := parse.ParseYAMLString(tt.Data)
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
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": "something",
			}),
		},
		{
			Name: "secret scalar",
			Data: yaml(`
				aws:
					accessKeyID: foo
					secretAccessKey: !Secret secretAccessKey
				op: something
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": testutil.JSONSecret("secretAccessKey"),
				},
				"op": "something",
			}),
		},
		{
			Name: "secret sequence",
			Data: yaml(`
				aws:
					accessKeyID: foo
					secretAccessKey: !Secret [secretAccessKey]
				op: something
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": testutil.JSONSecret("secretAccessKey"),
				},
				"op": "something",
			}),
		},
		{
			Name: "secret mapping",
			Data: yaml(`
				aws:
					accessKeyID: foo
					secretAccessKey: !Secret {name: secretAccessKey}
				op: something
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": testutil.JSONSecret("secretAccessKey"),
				},
				"op": "something",
			}),
		},
		{
			Name: "connection sequence",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Connection [aws, test-creds]
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": testutil.JSONConnection("aws", "test-creds"),
			}),
		},
		{
			Name: "connection mapping",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Connection {type: aws, name: test-creds}
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": testutil.JSONConnection("aws", "test-creds"),
			}),
		},
		{
			Name: "output sequence",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Output [prev, operation]
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": testutil.JSONOutput("prev", "operation"),
			}),
		},
		{
			Name: "output mapping",
			Data: yaml(`
				aws:
					accessKeyID: foo
					region: us-west-2
				op: !Output {from: prev, name: operation}
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": "foo",
					"region":      "us-west-2",
				},
				"op": testutil.JSONOutput("prev", "operation"),
			}),
		},
		{
			Name: "parameter scalar",
			Data: yaml(`
				aws:
					accessKeyID: !Parameter accessKeyID
					region: us-west-2
				op: something
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": testutil.JSONParameter("accessKeyID"),
					"region":      "us-west-2",
				},
				"op": "something",
			}),
		},
		{
			Name: "parameter sequence",
			Data: yaml(`
				aws:
					accessKeyID: !Parameter [accessKeyID]
					region: us-west-2
				op: something
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": testutil.JSONParameter("accessKeyID"),
					"region":      "us-west-2",
				},
				"op": "something",
			}),
		},
		{
			Name: "parameter mapping",
			Data: yaml(`
				aws:
					accessKeyID: !Parameter {name: accessKeyID}
					region: us-west-2
				op: something
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": testutil.JSONParameter("accessKeyID"),
					"region":      "us-west-2",
				},
				"op": "something",
			}),
		},
		{
			Name: "conditional invocation",
			Data: yaml(`
				- !Fn.equals [!Parameter param1, "foobar"]
				- !Fn.notEquals [!Parameter param2, "barfoo"]
				`),
			ExpectedTree: parse.Tree([]interface{}{
				testutil.JSONInvocation("equals", []interface{}{
					testutil.JSONParameter("param1"), "foobar",
				}),
				testutil.JSONInvocation("notEquals", []interface{}{
					testutil.JSONParameter("param2"), "barfoo",
				}),
			}),
		},
		{
			Name: "invocation scalar",
			Data: yaml(`
				message: !Fn.jsonUnmarshal '{"foo": "bar"}'
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"message": testutil.JSONInvocation("jsonUnmarshal", []interface{}{`{"foo": "bar"}`}),
			}),
		},
		{
			Name: "invocation sequence",
			Data: yaml(`
				message: !Fn.concat ["Hello, ", !Parameter name]
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"message": testutil.JSONInvocation("concat", []interface{}{
					"Hello, ",
					testutil.JSONParameter("name"),
				}),
			}),
		},
		{
			Name: "invocation mapping",
			Data: yaml(`
				message: !Fn.getMessage {host: api.example.com}
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"message": testutil.JSONInvocation("getMessage", map[string]interface{}{"host": "api.example.com"}),
			}),
		},
		{
			Name: "binary encoding",
			Data: yaml(`
				message: !!binary SGVsbG8sIJCiikU=
			`),
			ExpectedTree: parse.Tree(map[string]interface{}{
				"message": testutil.JSONEncoding("base64", "SGVsbG8sIJCiikU="),
			}),
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
