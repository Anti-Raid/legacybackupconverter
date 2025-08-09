package converter

import (
	"bytes"
	"fmt"

	"github.com/anti-raid/legacybackupconverter/iblfile"
	"github.com/vmihailenco/msgpack/v5"
)

func readMsgpackSection[T any](f *iblfile.AutoEncryptedFile_FullFile, name string) (*T, error) {
	section, err := f.Get(name)

	if err != nil {
		return nil, fmt.Errorf("failed to get section %s: %w", name, err)
	}

	dec := msgpack.NewDecoder(bytes.NewReader(section.Bytes()))
	dec.UseInternedStrings(true)
	dec.SetCustomStructTag("json")

	var outp T

	err = dec.Decode(&outp)

	if err != nil {
		return nil, fmt.Errorf("failed to decode section %s: %w", name, err)
	}

	return &outp, nil
}
