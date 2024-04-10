package credentialhandler

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
)

// ReadProperties implements parsing a stream formatted as defined by the Git
// credential storage documentation. Multi-valued properties are not
// implemented -- not required for this system.
//
// Note that the Reader may be fully consumed by this method.
//
// See also: https://git-scm.com/docs/git-credential#IOFMT
func ReadProperties(r io.Reader) (map[string]string, error) {
	pairs := map[string]string{}

	s := bufio.NewScanner(r)

	for s.Scan() {
		line := s.Text()

		if line == "" {
			// empty line terminates input
			break
		}

		// must have a delimiter, key must be valued, skip invalid
		k, v, ok := strings.Cut(line, "=")
		if !ok || k == "" {
			continue
		}

		pairs[k] = v
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	return pairs, nil
}

func WriteProperties(props map[string]string, w io.Writer) error {
	b := bytes.Buffer{}

	for k, v := range props {
		if k == "" || strings.ContainsAny(k, "\n=\000") {
			return errors.New("key empty or contains invalid character")
		}
		if strings.ContainsAny(v, "\n\000") {
			return errors.New("value empty or contains invalid character")
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	// write after serialization, to avoid partial output
	b.WriteTo(w)

	return nil
}
