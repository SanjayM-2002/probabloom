package username

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var ErrInvalid = errors.New("invalid username")

// Normalize maps every way of writing a name to one canonical form, so the
// filter sees "Robin", " robin " and "ｒｏｂｉｎ" (fullwidth) as the same name.
// Which characters are allowed is a product decision and isn't enforced here.
func Normalize(raw string) (string, error) {
	s := norm.NFKC.String(strings.TrimSpace(raw))
	s = strings.ToLower(s)
	switch {
	case s == "":
		return "", fmt.Errorf("%w: empty", ErrInvalid)
	case strings.ContainsFunc(s, unicode.IsSpace):
		return "", fmt.Errorf("%w: contains whitespace", ErrInvalid)
	case strings.ContainsFunc(s, unicode.IsControl):
		return "", fmt.Errorf("%w: contains control characters", ErrInvalid)
	}
	return s, nil
}
