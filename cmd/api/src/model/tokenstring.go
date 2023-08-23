package model

import (
	"crypto/rand"
	"fmt"
	"hash/crc32"
	"io"
	"math/big"
	"strings"
)

func generateTokenValue(prng io.Reader) (string, error) {
	var val strings.Builder
	big_range := big.NewInt(62) // a-zA-Z0-9 gives a range of 62 chars
	val.Grow(64)
	for i := 0; i < 64; i++ {
		if n, err := rand.Int(prng, big_range); err != nil {
			return "", fmt.Errorf("error getting random value: %w", err)
		} else {
			if _, err := val.WriteString(n.Text(62)); err != nil {
				return "", fmt.Errorf("error appending character to string: %w", err)
			}
		}
	}
	return val.String(), nil
}

type TokenString struct {
	Prefix string
	value  string
	cksum  uint32
}

func GenerateTokenString(prefix string) (TokenString, error) {
	if len(prefix) <= 0 {
		return TokenString{}, fmt.Errorf("prefix must not be empty")
	}
	if val, err := generateTokenValue(rand.Reader); err != nil {
		return TokenString{}, fmt.Errorf("error generating token string value: %w", err)
	} else {
		return CreateTokenStringWithValue(prefix, val)
	}
}

func CreateTokenStringWithValue(prefix, value string) (TokenString, error) {
	if len(prefix) <= 0 {
		return TokenString{}, fmt.Errorf("prefix must not be empty")
	}
	if len(value) != 64 {
		return TokenString{}, fmt.Errorf("value must be of length 64")
	}
	new := TokenString{Prefix: strings.ToUpper(prefix), value: value}
	new.cksum = crc32.ChecksumIEEE([]byte(new.Prefix + new.value))
	return new, nil
}

func formatChecksum(cksum uint32) string {
	return strings.ReplaceAll(fmt.Sprintf("%6s", big.NewInt(int64(cksum)).Text(62)), " ", "0")
}

func (s TokenString) String() string {
	if s.value == "" {
		return ""
	} else {
		return fmt.Sprintf("%s_%s%s", s.Prefix, s.value, formatChecksum(s.cksum))
	}
}

func (s TokenString) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func isValidBase62(val string) bool {
	for _, v := range []byte(val) {
		if v < '0' || (v > '9' && v < 'A') || (v > 'Z' && v < 'a') || v > 'z' {
			return false
		}
	}
	return true
}
