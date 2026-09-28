package stigma

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

// WAL is a simple append-only log of samples.
type WAL struct {
	path string
	file *os.File
	enc  *json.Encoder
	mu   sync.Mutex
}

// OpenWAL opens or creates the WAL at the path.
func OpenWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	// move to end for appends
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		return nil, err
	}

	return &WAL{path: path, file: file, enc: json.NewEncoder(file)}, nil
}

// Append writes samples to the log.
func (w *WAL) Append(samples []LocationSample) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, s := range samples {
		if err := w.enc.Encode(&s); err != nil {
			return err
		}
	}
	return nil
}

// ReadAll replays all entries from the beginning.
func (w *WAL) ReadAll() ([]LocationSample, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(w.file)
	var entries []LocationSample
	for {
		var s LocationSample
		if err := dec.Decode(&s); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		entries = append(entries, s)
	}

	// return file to end for future appends
	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	return entries, nil
}

// Close closes the WAL file.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
