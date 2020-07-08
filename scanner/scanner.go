// scanner returns URLs, and should be used like bufio.Scanner, except that the Err should be checked each loop.
package scanner

import (
	"bufio"
	"errors"
	"io"
	"net/url"
	"strings"
)

func parseURL(u string) (*url.URL, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, errors.New(u) // Error text is just the URL
	}
	// Add scheme to URLs for convenience, so that you can write a command like: gemget example.com
	// instead of: gemget gemini://example.com
	if !strings.HasPrefix(u, "//") && !strings.Contains(u, "://") {
		tmp := "gemini://" + u
		parsed, err = url.Parse(tmp)
		if err != nil {
			return nil, errors.New(tmp)
		}
	}
	return parsed, nil
}

type Scanner struct {
	bufScanner *bufio.Scanner
	urls       []string // Initally passed URLs - not modified later on
	current    *url.URL
	err        error
	n          int // The number of URLs processed so far
}

// NewScanner returns a new Scanner that treats r like a file.
// The URL strings passed, if any, will be processed first.
//
// r can be nil.
func NewScanner(r io.Reader, urls ...string) *Scanner {
	if r == nil {
		return &Scanner{bufScanner: nil, urls: urls}
	}
	return &Scanner{bufScanner: bufio.NewScanner(r), urls: urls}
}

func (s *Scanner) Scan() bool {
	defer func() {
		s.n++
	}()

	// Reset each time
	s.err = nil
	s.current = nil

	if s.n < len(s.urls) {
		// Need to go through passed URL strings first (CLI args)
		s.current, s.err = parseURL(s.urls[s.n])
		return true
	} else if s.bufScanner == nil {
		// Done with URL strings, but there's nothing to read from
		return false
	} else {
		// There's a reader, and we're done with URL strings
		ok := s.bufScanner.Scan()
		if !ok {
			// No more URLs
			s.err = s.bufScanner.Err()
			return false
		}
		// More URLs in the file

		// Keep scanning until there's a non-empty, non-comment line
		for strings.TrimSpace(s.bufScanner.Text()) == "" || strings.HasPrefix(strings.TrimLeft(s.bufScanner.Text(), " \t"), "#") {
			ok := s.bufScanner.Scan()
			if !ok {
				// No more URLs
				s.err = s.bufScanner.Err()
				return false
			}
		}
		// Found a potential URL
		s.current, s.err = parseURL(strings.TrimSpace(s.bufScanner.Text()))
		return true
	}
}

func (s *Scanner) URL() *url.URL {
	// Return a copy bc this is modified internally
	tmp := *s.current
	return &tmp
}

func (s *Scanner) Err() error {
	return s.err
}
