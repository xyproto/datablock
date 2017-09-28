package datablock

import (
	"bytes"
	"compress/gzip"
	"github.com/mattetti/filebuffer"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// DataBlock represents a block of data that may be compressed
type DataBlock struct {
	data             []byte
	compressed       bool
	length           int
	compressionSpeed bool // prefer speed over best compression ratio?
}

var (
	// EmptyDataBlock is an empty data block
	EmptyDataBlock = &DataBlock{[]byte{}, false, 0, true}
)

// NewDataBlock creates a new uncompressed data block.
// compressionSpeed is if speedy compression should be used over compact compression
func NewDataBlock(data []byte, compressionSpeed bool) *DataBlock {
	return &DataBlock{data, false, len(data), compressionSpeed}
}

// Create a new data block where the data may already be compressed.
// compressionSpeed is if speedy compression should be used over compact compression
func newDataBlockSpecified(data []byte, compressed bool, compressionSpeed bool) *DataBlock {
	return &DataBlock{data, compressed, len(data), compressionSpeed}
}

// UncompressedData returns the the original, uncompressed data,
// the length of the data and an error. Will decompress if needed.
func (b *DataBlock) UncompressedData() ([]byte, int, error) {
	if b.compressed {
		return decompress(b.data)
	}
	return b.data, b.length, nil
}

// MustData returns the uncompressed data or an empty byte slice
func (b *DataBlock) MustData() []byte {
	if b.compressed {
		data, _, err := decompress(b.data)
		if err != nil {
			log.Fatal(err)
			return []byte{}
		}
		return data
	}
	return b.data
}

// String returns the uncompressed data as a string or as an empty string.
// Same as MustData, but converted to a string.
func (b *DataBlock) String() string {
	return string(b.MustData())
}

// Gzipped returns the compressed data, length and an error.
// Will compress if needed.
func (b *DataBlock) Gzipped() ([]byte, int, error) {
	if !b.compressed {
		return compress(b.data, b.compressionSpeed)
	}
	return b.data, b.length, nil
}

// Compress this data block
func (b *DataBlock) Compress() error {
	if b.compressed {
		return nil
	}
	data, bytesWritten, err := compress(b.data, b.compressionSpeed)
	if err != nil {
		return err
	}
	b.data = data
	b.compressed = true
	b.length = bytesWritten
	return nil
}

// Decompress this data block
func (b *DataBlock) Decompress() error {
	if !b.compressed {
		return nil
	}
	data, bytesWritten, err := decompress(b.data)
	if err != nil {
		return err
	}
	b.data = data
	b.compressed = false
	b.length = bytesWritten
	return nil
}

// IsCompressed checks if this data block is compressed
func (b *DataBlock) IsCompressed() bool {
	return b.compressed
}

// StringLength returns the length of the data, represented as a string
func (b *DataBlock) StringLength() string {
	return strconv.Itoa(b.length)
}

// Length returns the lentgth of the current data
// (not the length of the original data, but in the current state)
func (b *DataBlock) Length() int {
	return b.length
}

// HasData returns true if there is data present
func (b *DataBlock) HasData() bool {
	return 0 != b.length
}

// ToClient writes the data to the client.
// Also sets the right headers and compresses the data with gzip if needed.
// Set canGzip to true if the http client can handle gzipped data.
// gzipThreshold is the threshold (in bytes) for when it makes sense to compress the data with gzip
func (b *DataBlock) ToClient(w http.ResponseWriter, req *http.Request, name string, canGzip bool, gzipThreshold int) {
	overThreshold := b.Length() > gzipThreshold // Is there enough data that it makes sense to compress it?

	// Compress or decompress the data as needed. Add headers if compression is used.
	if !canGzip {
		// No compression
		if err := b.Decompress(); err != nil {
			// Unable to decompress gzipped data!
			log.Fatal(err)
		}
	} else if b.compressed || overThreshold {
		// If the given data is already compressed, or we are planning to compress,
		// set the gzip headers and serve it as compressed data.

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")

		// If the data is over a certain size, compress and serve
		if overThreshold {
			// Compress
			if err := b.Compress(); err != nil {
				// Write uncompressed data if gzip should fail
				log.Error(err)
				w.Header().Set("Content-Encoding", "identity")
			}
		}
	}

	// Done by ServeContent instead
	//w.Header().Set("Content-Length", b.StringLength())
	//w.Write(b.data)

	// Serve the data with http.ServeContent, which supports ranges/streaming
	http.ServeContent(w, req, name, time.Time{}, filebuffer.New(b.data))
}

// Compress data using gzip. Returns the data, data length and an error.
func compress(data []byte, speed bool) ([]byte, int, error) {
	if len(data) == 0 {
		return []byte{}, 0, nil
	}
	var buf bytes.Buffer
	_, err := gzipWrite(&buf, data, speed)
	if err != nil {
		return nil, 0, err
	}
	data = buf.Bytes()
	return data, len(data), nil
}

// Decompress data using gzip. Returns the data, data length and an error.
func decompress(data []byte) ([]byte, int, error) {
	if len(data) == 0 {
		return []byte{}, 0, nil
	}
	var buf bytes.Buffer
	_, err := gunzipWrite(&buf, data)
	if err != nil {
		return nil, 0, err
	}
	data = buf.Bytes()
	return data, len(data), nil
}

// Write gzipped data to a Writer. Returns bytes written and an error.
func gzipWrite(w io.Writer, data []byte, speed bool) (int, error) {
	// Write gzipped data to the client
	level := gzip.BestCompression
	if speed {
		level = gzip.BestSpeed
	}
	gw, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return 0, err
	}
	defer gw.Close()
	bytesWritten, err := gw.Write(data)
	if err != nil {
		return 0, err
	}
	return bytesWritten, nil
}

// Write gunzipped data to a Writer. Returns bytes written and an error.
func gunzipWrite(w io.Writer, data []byte) (int, error) {
	// Write gzipped data to the client
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return 0, err
	}
	bytesWritten, err := w.Write(data)
	if err != nil {
		return 0, err
	}
	return bytesWritten, nil
}
