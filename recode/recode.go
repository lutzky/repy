package recode

import (
	"github.com/pkg/errors"
	"golang.org/x/text/encoding/charmap"
)

func Recode(from, to *charmap.Charmap, data []byte) ([]byte, error) {
	dataUTF8, err := from.NewDecoder().Bytes(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert from %s to UTF-8", from.String())
	}
	return to.NewEncoder().Bytes(dataUTF8)
}
