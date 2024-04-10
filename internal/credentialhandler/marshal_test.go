package credentialhandler_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/jamestelfer/ghauth/internal/credentialhandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadProperties(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty",
			input:    "",
			expected: map[string]string{},
		},
		{
			name: "stop at empty line",
			input: `
				one=1
				
				three=3
			`,
			expected: map[string]string{
				"one": "1",
			},
		},
		{
			name: "handle empty values",
			input: `
				one=1
				two=
				three=3
			`,
			expected: map[string]string{
				"one":   "1",
				"two":   "",
				"three": "3",
			},
		},
		{
			name: "skip empty keys",
			input: `
				one=1
				=2
				three=3
			`,
			expected: map[string]string{
				"one":   "1",
				"three": "3",
			},
		},
		{
			name: "skip missing delimiter",
			input: `
				one
				two=2
				three
			`,
			expected: map[string]string{
				"two": "2",
			},
		},
		{
			name: "normal",
			input: `
				one=1
				two=2
				three=3
			`,
			expected: map[string]string{
				"one":   "1",
				"two":   "2",
				"three": "3",
			},
		},
	}

	leadingSpace := regexp.MustCompile(`(?m)^[\t ]*?(\b|$|\z|\S)`)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := leadingSpace.ReplaceAllString(c.input, "$1")
			input = strings.TrimPrefix(input, "\n")
			input = strings.TrimSuffix(input, "\n")

			r := strings.NewReader(input)
			actual, err := credentialhandler.ReadProperties(r)
			require.NoError(t, err)

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestWriteProperties(t *testing.T) {
	cases := []struct {
		name     string
		input    map[string]string
		expected string
		failed   bool
	}{
		{
			name:     "empty",
			input:    map[string]string{},
			expected: "",
		},
		{
			name: "handle empty values",
			input: map[string]string{
				"one":   "1",
				"two":   "",
				"three": "3",
			},
			expected: `
				one=1
				two=
				three=3
			`,
		},
		{
			name: "fail empty keys",
			input: map[string]string{
				"one":   "1",
				"":      "2",
				"three": "3",
			},
			expected: "",
			failed:   true,
		},
		{
			name: "fail invalid key with '\\n'",
			input: map[string]string{
				"one\n": "1",
			},
			expected: "",
			failed:   true,
		},
		{
			name: "fail invalid key with '\\0'",
			input: map[string]string{
				"o\000ne": "1",
			},
			expected: "",
			failed:   true,
		},
		{
			name: "fail invalid key with '='",
			input: map[string]string{
				"on=e": "1",
			},
			expected: "",
			failed:   true,
		},
		{
			name: "fail invalid value with \\n",
			input: map[string]string{
				"one": "1\n",
			},
			expected: "",
			failed:   true,
		},
		{
			name: "fail invalid value with \\0",
			input: map[string]string{
				"one": "\0001",
			},
			expected: "",
			failed:   true,
		},
		{
			name: "value with '='",
			input: map[string]string{
				"one": "=1=",
			},
			expected: `
				one==1=
			`,
		},
		{
			name: "normal",
			input: map[string]string{
				"one":   "1",
				"two":   "2",
				"three": "3",
			},
			expected: `
				one=1
				two=2
				three=3
			`,
		},
	}

	leadingSpace := regexp.MustCompile(`(?m)^[\t ]*?(\b|$|\z|\S)`)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expected := leadingSpace.ReplaceAllString(c.expected, "$1")
			expected = strings.TrimPrefix(expected, "\n")

			w := &bytes.Buffer{}

			err := credentialhandler.WriteProperties(c.input, w)
			require.Equal(t, err != nil, c.failed, "%v", err)

			assert.Equal(t, expected, w.String())
		})
	}
}
