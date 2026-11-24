package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	cerrors "shorter/core/errors"
	"shorter/core/models"
	"shorter/core/ports"
)

const (
	maxAttempts            = 5
	decimalBase            = 10
	portBitSize            = 16
	ipv4OctetCount         = 4
	octalBase              = 8
	hexadecimalBase        = 16
	bitsPerByte            = 8
	base64BitsPerCharacter = 6
	codeEntropyBytes       = models.CodeLength * base64BitsPerCharacter / bitsPerByte
)

type LinkService struct {
	repository  ports.LinkRepository
	generator   ports.CodeGenerator
	baseURL     string
	codePattern *regexp.Regexp
}

func NewLinkService(repository ports.LinkRepository, generator ports.CodeGenerator, baseURL string) (*LinkService, error) {
	u, err := parseURL(baseURL)
	if err != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(baseURL, "#") {
		return nil, fmt.Errorf("invalid BASE_URL: %w", cerrors.ErrInvalidURL)
	}
	return &LinkService{
		repository:  repository,
		generator:   generator,
		baseURL:     strings.TrimRight(baseURL, "/"),
		codePattern: regexp.MustCompile(fmt.Sprintf(`\A[A-Za-z0-9_-]{%d}\z`, models.CodeLength)),
	}, nil
}

func (s *LinkService) Shorten(ctx context.Context, originalURL string) (string, error) {
	if _, err := parseURL(originalURL); err != nil {
		return "", cerrors.ErrInvalidURL
	}
	for range maxAttempts {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		code, err := s.generator.Generate()
		if err != nil {
			return "", fmt.Errorf("generate short code: %w", err)
		}
		err = s.repository.Save(ctx, models.Link{Code: code, URL: originalURL})
		if errors.Is(err, cerrors.ErrCodeConflict) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("save link: %w", err)
		}
		return s.baseURL + "/" + code, nil
	}
	return "", fmt.Errorf("short code attempts exhausted: %w", cerrors.ErrCodeConflict)
}

func (s *LinkService) Resolve(ctx context.Context, code string) (string, error) {
	if !s.codePattern.MatchString(code) {
		return "", cerrors.ErrInvalidCode
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	link, err := s.repository.FindByCode(ctx, code)
	if err != nil {
		return "", fmt.Errorf("find link: %w", err)
	}
	return link.URL, nil
}

type Random struct{}

func (Random) Generate() (string, error) {
	var value [codeEntropyBytes]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

func parseURL(value string) (*url.URL, error) {
	if len(value) == 0 || len(value) > models.MaxURLBytes {
		return nil, cerrors.ErrInvalidURL
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return nil, cerrors.ErrInvalidURL
	}
	if !strings.HasPrefix(u.Host, "[") && !validNumericHost(u.Hostname()) {
		return nil, cerrors.ErrInvalidURL
	}
	if port := u.Port(); port != "" {
		number, err := strconv.ParseUint(port, decimalBase, portBitSize)
		if err != nil || number == 0 {
			return nil, cerrors.ErrInvalidURL
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return nil, cerrors.ErrInvalidURL
	}
	return u, nil
}

// validNumericHost checks IPv4 syntax for hosts that browsers interpret as IPv4.
// See https://url.spec.whatwg.org/#concept-host-parser.
func validNumericHost(host string) bool {
	host = strings.TrimSuffix(host, ".")
	lastLabel := host[strings.LastIndexByte(host, '.')+1:]
	if !endsInIPv4Number(lastLabel) {
		return true
	}

	parts := strings.Split(host, ".")
	if len(parts) > ipv4OctetCount {
		return false
	}
	for i, part := range parts {
		bitSize := bitsPerByte
		if i == len(parts)-1 {
			// In abbreviated IPv4 addresses, the last part fills the remaining octets.
			bitSize = (ipv4OctetCount - i) * bitsPerByte
		}
		if !validIPv4Part(part, bitSize) {
			return false
		}
	}
	return true
}

func endsInIPv4Number(label string) bool {
	if hasHexPrefix(label) {
		return strings.Trim(label[2:], "0123456789abcdefABCDEF") == ""
	}
	return label != "" && strings.Trim(label, "0123456789") == ""
}

func validIPv4Part(part string, bitSize int) bool {
	base := decimalBase
	switch {
	case hasHexPrefix(part):
		base = hexadecimalBase
		part = part[2:]
		if part == "" {
			return true // The browser parser interprets a bare 0x prefix as zero.
		}
	case len(part) > 1 && part[0] == '0':
		base = octalBase
	}
	_, err := strconv.ParseUint(part, base, bitSize)
	return err == nil
}

func hasHexPrefix(value string) bool {
	return strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X")
}
