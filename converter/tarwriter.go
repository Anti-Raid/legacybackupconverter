package converter

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
)

type SourceParsed struct {
	Data  map[string]any
	Table string
}

type TarFile struct {
	tarWriter *tar.Writer
	buf       *bytes.Buffer
}

// Returns the size of the file
func (f *TarFile) Size() int {
	return f.buf.Len()
}

func NewTarFile() *TarFile {
	buf := bytes.NewBuffer([]byte{})
	tarWriter := tar.NewWriter(buf)

	return &TarFile{
		buf:       buf,
		tarWriter: tarWriter,
	}
}

// Adds a section to a file
func (f *TarFile) WriteSection(buf *bytes.Buffer, name string) error {
	err := f.tarWriter.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0600,
		Size: int64(buf.Len()),
	})

	if err != nil {
		return err
	}

	_, err = f.tarWriter.Write(buf.Bytes())

	if err != nil {
		return err
	}

	return nil
}

// Adds a section to a file with json file format
func (f *TarFile) WriteJsonGzSection(i any, name string) error {
	buf := bytes.NewBuffer([]byte{})

	err := json.NewEncoder(buf).Encode(i)

	if err != nil {
		return err
	}

	// Gzip the buffer
	gzippedBuf := bytes.NewBuffer([]byte{})
	gzWriter := gzip.NewWriter(gzippedBuf)
	_, err = gzWriter.Write(buf.Bytes())
	if err != nil {
		return err
	}
	err = gzWriter.Close()
	if err != nil {
		return err
	}
	return f.WriteSection(gzippedBuf, name)
}

func (f *TarFile) Build() (*bytes.Buffer, error) {
	// Close tar file
	err := f.tarWriter.Close()

	if err != nil {
		return nil, err
	}

	// Return the buffer
	return f.buf, nil
}
