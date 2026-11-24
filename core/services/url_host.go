package services

import (
	"strconv"
	"strings"
)

const (
	ipv4OctetCount  = 4
	bitsPerOctet    = 8
	octalBase       = 8
	hexadecimalBase = 16
)

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
		bitSize := bitsPerOctet
		if i == len(parts)-1 {
			// In abbreviated IPv4 addresses, the last part fills the remaining octets.
			bitSize = (ipv4OctetCount - i) * bitsPerOctet
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
