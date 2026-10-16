package registry

import (
	"bytes"
	"context"
	"encoding"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SanjayM-2002/probabloom"
)

// File layout:
//
//	"PBLM" | version (1 byte) | header length (4 bytes, LE) | JSON Config | filter payload
//
// The header records everything needed to rebuild a filter of the same
// shape; the payload is then loaded into it. Loading with different settings
// would put the bits in the wrong places, so the header is the only source
// of truth for a saved filter.
const (
	fileMagic   = "PBLM"
	fileVersion = 1
)

var ErrNotSavable = errors.New("this filter can't be saved to a file (only the memory backend can)")

func Save(w io.Writer, cfg Config, f probabloom.Filter) error {
	m, ok := f.(encoding.BinaryMarshaler)
	if !ok || cfg.Backend != "memory" {
		return ErrNotSavable
	}
	header, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	payload, err := m.MarshalBinary()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.WriteString(fileMagic)
	buf.WriteByte(fileVersion)
	buf.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(header))))
	buf.Write(header)
	buf.Write(payload)
	_, err = w.Write(buf.Bytes())
	return err
}

func Load(ctx context.Context, r io.Reader) (probabloom.Filter, Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, Config{}, err
	}
	if len(data) < 9 || string(data[:4]) != fileMagic {
		return nil, Config{}, fmt.Errorf("not a probabloom file")
	}
	if data[4] != fileVersion {
		return nil, Config{}, fmt.Errorf("unsupported file version %d", data[4])
	}
	hlen := binary.LittleEndian.Uint32(data[5:9])
	if uint64(len(data)) < 9+uint64(hlen) {
		return nil, Config{}, fmt.Errorf("truncated header")
	}

	var cfg Config
	if err := json.Unmarshal(data[9:9+hlen], &cfg); err != nil {
		return nil, Config{}, fmt.Errorf("bad header: %w", err)
	}
	f, err := New(ctx, cfg)
	if err != nil {
		return nil, Config{}, err
	}
	u, ok := f.(encoding.BinaryUnmarshaler)
	if !ok {
		return nil, Config{}, ErrNotSavable
	}
	if err := u.UnmarshalBinary(data[9+hlen:]); err != nil {
		return nil, Config{}, err
	}
	return f, cfg, nil
}

// SaveFile writes to a temp file and renames it, so a crash mid-write never
// leaves a half-written filter behind.
func SaveFile(path string, cfg Config, f probabloom.Filter) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".probabloom-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := Save(tmp, cfg, f); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func LoadFile(ctx context.Context, path string) (probabloom.Filter, Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, Config{}, err
	}
	defer file.Close()
	return Load(ctx, file)
}
