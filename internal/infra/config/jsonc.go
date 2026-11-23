package config

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func jsonc(raw []byte) ([]byte, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	clean := append([]byte(nil), raw...)
	quoted := false
	for i := 0; i < len(clean); i++ {
		switch {
		case quoted && clean[i] == '\\':
			i++
		case clean[i] == '"':
			quoted = !quoted
		case !quoted && clean[i] == '/' && i+1 < len(clean):
			end := i
			switch clean[i+1] {
			case '/':
				end = i + 2
				for end < len(clean) && clean[end] != '\n' && clean[end] != '\r' {
					end++
				}
			case '*':
				offset := bytes.Index(clean[i+2:], []byte("*/"))
				if offset < 0 {
					return nil, fmt.Errorf("unterminated block comment")
				}
				end = i + 4 + offset
			}
			if end > i {
				for j := i; j < end; j++ {
					if clean[j] != '\r' && clean[j] != '\n' {
						clean[j] = ' '
					}
				}
				i = end - 1
			}
		}
	}
	var previous byte
	for i := 0; i < len(clean); i++ {
		switch {
		case quoted && clean[i] == '\\':
			i++
		case clean[i] == '"':
			quoted = !quoted
			previous = '"'
		case !quoted && clean[i] == ',':
			next := bytes.TrimLeft(clean[i+1:], " \r\n\t")
			if len(next) > 0 && (next[0] == '}' || next[0] == ']') && previous != '{' && previous != '[' && previous != ',' && previous != ':' {
				clean[i] = ' '
			}
			previous = ','
		case !quoted && !bytes.ContainsRune([]byte(" \r\n\t"), rune(clean[i])):
			previous = clean[i]
		}
	}
	if !json.Valid(clean) {
		var value json.RawMessage
		return nil, json.Unmarshal(clean, &value)
	}
	return clean, nil
}
