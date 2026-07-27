// Package waid menormalkan nomor WhatsApp ke satu bentuk kanonik.
//
// Bentuk kanonik = E.164 tanpa "+", mis. "6281231372105".
// Ini satu-satunya sumber kebenaran: WAHA mengirim JID "628...@c.us",
// sedangkan admin mengetik "0812...". Tanpa normalisasi di kedua sisi,
// guru yang sudah terdaftar akan ditolak sebagai "belum terdaftar".
package waid

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalid dikembalikan bila input tidak bisa ditafsirkan sebagai nomor
// WhatsApp Indonesia yang wajar.
var ErrInvalid = errors.New("nomor WhatsApp tidak valid")

var nonDigit = regexp.MustCompile(`\D`)

// Normalize menerima "0812...", "+62 812-...", "812...", atau JID WAHA
// "628...@c.us" dan mengembalikan bentuk kanonik "628...".
func Normalize(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", ErrInvalid
	}

	// Buang suffix JID: "@c.us", "@s.whatsapp.net", "@g.us", dan device suffix ":12@c.us".
	atIdx := strings.IndexByte(s, '@')
	colonIdx := strings.IndexByte(s, ':')
	var idx int
	if atIdx >= 0 && colonIdx >= 0 {
		idx = atIdx
		if colonIdx < atIdx {
			idx = colonIdx
		}
	} else if atIdx >= 0 {
		idx = atIdx
	} else if colonIdx >= 0 {
		idx = colonIdx
	} else {
		idx = -1
	}
	if idx >= 0 {
		s = s[:idx]
	}

	s = nonDigit.ReplaceAllString(s, "")

	switch {
	case strings.HasPrefix(s, "62"):
		// sudah memakai kode negara
	case strings.HasPrefix(s, "0"):
		s = "62" + s[1:]
	case strings.HasPrefix(s, "8"):
		// admin kadang menulis tanpa 0 di depan
		s = "62" + s
	default:
		return "", ErrInvalid
	}

	if len(s) < 10 || len(s) > 15 {
		return "", ErrInvalid
	}
	return s, nil
}
