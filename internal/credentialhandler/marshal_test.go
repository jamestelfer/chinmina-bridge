package credentialhandler_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/jamestelfer/chinmina-bridge/internal/credentialhandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadProperties(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected [][]string
	}{
		{
			name:     "nil handling",
			input:    "~~nil~~",
			expected: [][]string{},
		},
		{
			name:     "empty",
			input:    "",
			expected: [][]string{},
		},
		{
			name: "stop at empty line",
			input: `
one=1

three=3
`,
			expected: [][]string{
				{"one", "1"},
			},
		},
		{
			name: "handle empty values",
			input: `
one=1
two=
three=3
`,
			expected: [][]string{
				{"one", "1"},
				{"two", ""},
				{"three", "3"},
			},
		},
		{
			name: "skip empty keys",
			input: `
one=1
=2
three=3
`,
			expected: [][]string{
				{"one", "1"},
				{"three", "3"},
			},
		},
		{
			name: "skip missing delimiter",
			input: `
one
two=2
three
`,
			expected: [][]string{
				{"two", "2"},
			},
		},
		{
			name: "normal",
			input: `
one=1
two=2
three=3
`,
			expected: [][]string{
				{"one", "1"},
				{"two", "2"},
				{"three", "3"},
			},
		},
	}

	inputReader := func(input string) io.Reader {
		if input == "~~nil~~" { // something of a hack but easier than making it a pointer.
			return nil
		}

		input = strings.TrimPrefix(input, "\n")
		return strings.NewReader(input)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := inputReader(c.input)
			actual, err := credentialhandler.ReadProperties(r)
			require.NoError(t, err)

			expected := credentialhandler.NewMapFromArray(c.expected)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestWriteProperties(t *testing.T) {
	cases := []struct {
		name     string
		input    [][]string
		expected []string
		failed   bool
	}{
		{
			name:     "empty",
			input:    [][]string{},
			expected: []string{"", ""},
		},
		{
			name: "handle empty values",
			input: [][]string{
				{"one", "1"},
				{"two", ""},
				{"three", "3"},
			},
			expected: []string{"one=1", "two=", "three=3", "", ""},
		},
		{
			name: "fail empty keys",
			input: [][]string{
				{"one", "1"},
				{"", "2"},
				{"three", "3"},
			},
			expected: []string{""},
			failed:   true,
		},
		{
			name: "fail invalid key with '\\n'",
			input: [][]string{
				{"one\n", "1"},
			},
			expected: []string{""},
			failed:   true,
		},
		{
			name: "fail invalid key with '\\0'",
			input: [][]string{
				{"o\000ne", "1"},
			},
			expected: []string{""},
			failed:   true,
		},
		{
			name: "fail invalid key with '='",
			input: [][]string{
				{"on=e", "1"},
			},
			expected: []string{""},
			failed:   true,
		},
		{
			name: "fail invalid value with \\n",
			input: [][]string{
				{"one", "1\n"},
			},
			expected: []string{""},
			failed:   true,
		},
		{
			name: "fail invalid value with \\0",
			input: [][]string{
				{"one", "\0001"},
			},
			expected: []string{""},
			failed:   true,
		},
		{
			name: "value with '='",
			input: [][]string{
				{"one", "=1="},
			},
			expected: []string{"one==1=", "", ""},
		},
		{
			name: "normal",
			input: [][]string{
				{"one", "1"},
				{"two", "2"},
				{"three", "3"},
			},
			expected: []string{"one=1", "two=2", "three=3", "", ""},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &bytes.Buffer{}

			input := credentialhandler.NewMapFromArray(c.input)
			err := credentialhandler.WriteProperties(input, w)
			require.Equal(t, err != nil, c.failed, "%v", err)

			assert.Equal(t, c.expected, strings.Split(w.String(), "\n"), w.String())
		})
	}
}
