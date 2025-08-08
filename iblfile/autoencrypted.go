package iblfile

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
)

var (
	AutoEncryptedFileMagic        = []byte("iblaef")
	AutoEncryptedFileChecksumSize = 32 // sha256
	AutoEncryptedFileIDSize       = 16
)

func AutoEncryptedMetadataSize() int {
	return len(AutoEncryptedFileMagic) + AutoEncryptedFileChecksumSize + AutoEncryptedFileIDSize
}

// Autoencrypted files can be encypted in many ways
//
// This defines an interface for all of them
type AutoEncryptor interface {
	// Returns the identifier of the source, must be unique
	//
	// Max size: 8 ASCII characters (8 bytes)
	ID() string
	// Decrypts a byte slice
	Decrypt([]byte) ([]byte, error) // Decrypts a byte slice
}

var AutoEncryptorRegistry = make(map[string]AutoEncryptor)

func RegisterAutoEncryptor(src AutoEncryptor) {
	id := []byte(src.ID())

	if len(id) != AutoEncryptedFileIDSize {
		panic(fmt.Errorf("invalid id size for %v: %v", src.ID(), len(id)))
	}

	AutoEncryptorRegistry[string(id)] = src
}

// Represents an autoencrypted file block
type AutoEncryptedFileBlock struct {
	// Magic bytes
	Magic []byte
	// Checksum
	Checksum []byte
	// Encryptor
	Encryptor []byte
	// Data
	Data []byte
}

// Validates a block to ensure that it is a valid autoencrypted file block
func (b *AutoEncryptedFileBlock) Validate() error {
	if string(b.Magic) != string(AutoEncryptedFileMagic) {
		return fmt.Errorf("invalid magic: %v", b.Magic)
	}

	// Calculate sha256 checksum of data
	checksum := sha256.Sum256(b.Data)

	if string(checksum[:]) != string(b.Checksum) {
		return fmt.Errorf("invalid checksum: %v", b.Checksum)
	}

	return nil
}

// Decrypts a block into a byte slice
func (b *AutoEncryptedFileBlock) Decrypt(src AutoEncryptor) ([]byte, error) {
	if src.ID() != string(b.Encryptor) {
		return nil, fmt.Errorf("invalid encryptor: %v", b.Encryptor)
	}

	return src.Decrypt(b.Data)
}

func ParseAutoEncryptedFileBlock(block []byte) (*AutoEncryptedFileBlock, error) {
	if len(block) < AutoEncryptedMetadataSize() {
		return nil, fmt.Errorf("block is too small")
	}

	var currentPos int

	// Magic
	magic := block[currentPos : currentPos+len(AutoEncryptedFileMagic)]
	currentPos += len(AutoEncryptedFileMagic)

	// Checksum
	checksum := block[currentPos : currentPos+AutoEncryptedFileChecksumSize]
	currentPos += AutoEncryptedFileChecksumSize

	// Encryptor
	encryptor := block[currentPos : currentPos+AutoEncryptedFileIDSize]
	currentPos += AutoEncryptedFileIDSize

	// Data
	data := block[currentPos:]

	return &AutoEncryptedFileBlock{
		Magic:     magic,
		Checksum:  checksum,
		Encryptor: encryptor,
		Data:      data,
	}, nil
}

// QuickBlockParser reads the first AutoEncryptedMetadataSize into a buffer and parses it
//
// Note that the block returned by this is *not* valid and is only meant for quick parsing of the encryptor
func QuickBlockParser(r io.ReadSeeker) (*AutoEncryptedFileBlock, error) {
	// Read the first AutoEncryptedMetadataSize into a buffer
	// This is the metadata section
	buf := make([]byte, AutoEncryptedMetadataSize())
	_, err := r.Read(buf)

	if err != nil {
		return nil, fmt.Errorf("error reading metadata: %w", err)
	}

	// This metadata will be 'corrupt', but we just need the encryptor
	meta, err := ParseAutoEncryptedFileBlock(buf)

	if err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}

	// Seek back to start
	_, err = r.Seek(0, 0)

	if err != nil {
		return nil, fmt.Errorf("error seeking back to start of file: %w", err)
	}

	return meta, nil
}

// A full file autoencrypted file. This type stores all data as one single encrypted block rather than per-section blocks
//
// This is the first, and simplest+quickest autoencrypted () file
type AutoEncryptedFile_FullFile struct {
	src      AutoEncryptor
	file     *RawFile
	sections map[string]*bytes.Buffer
}

// OpenAutoEncryptedFile_FullFile opens a full file as a single autoencrypted  block
func OpenAutoEncryptedFile_FullFile(r io.Reader, src AutoEncryptor) (*AutoEncryptedFile_FullFile, error) {
	data, err := io.ReadAll(r)

	if err != nil {
		return nil, err
	}

	block, err := ParseAutoEncryptedFileBlock(data)

	if err != nil {
		return nil, err
	}

	if err := block.Validate(); err != nil {
		return nil, fmt.Errorf("block is not valid: %v", err)
	}

	decryptedBlock, err := block.Decrypt(src)

	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(decryptedBlock)
	tarWriter := tar.NewWriter(buf)

	return &AutoEncryptedFile_FullFile{
		src: src,
		file: &RawFile{
			buf:       buf,
			tarWriter: tarWriter,
		},
	}, nil
}

// Returns all sections of the file
func (f *AutoEncryptedFile_FullFile) Sections() (map[string]*bytes.Buffer, error) {
	if f.sections != nil {
		return f.sections, nil
	}

	if f.file.buf.Len() == 0 {
		return map[string]*bytes.Buffer{}, nil
	}

	// Now, we have a decrypted tar file
	files, err := ReadTarFile(f.file.buf)

	if err != nil {
		return nil, fmt.Errorf("failed to parse raw data: %w", err)
	}

	f.sections = files
	return files, nil
}

// Get a section from the file
func (f *AutoEncryptedFile_FullFile) Get(name string) (*bytes.Buffer, error) {
	sections, err := f.Sections()

	if err != nil {
		return nil, err
	}

	section, ok := sections[name]

	if !ok {
		return nil, fmt.Errorf("no section found for %s", name)
	}

	return section, nil
}

// Returns the size of the file
func (f *AutoEncryptedFile_FullFile) Size() int {
	return f.file.Size()
}
