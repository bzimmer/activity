package activity

//go:generate stringer -type=Format -linecomment -output=xfer_string.go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	pollInterval   = 2 * time.Second
	pollIterations = 5
)

var ErrExceededIterations = errors.New("exceeded iterations")

// Export the contents and metadata about an activity file
type Export struct {
	*File
	ID int64 `json:"id"`
}

// Exporter exports activity data by activity id
type Exporter interface {
	// Export exports the data file
	Export(ctx context.Context, activityID int64) (*Export, error)
}

// UploadID is the type for all upload identifiers
type UploadID int64

// Upload is the current status of an upload request
type Upload interface {
	// Identifier is unique id for the upload
	Identifier() UploadID
	// Done returns whether the upload is complete, either successfully or an error occurred
	Done() bool
}

// Poll is the result of polling
type Poll struct {
	// Upload is the upload status if no error occurred
	Upload Upload
	// Err is non-nil when an error occurred in the operation but not semantically
	// Check the `Upload` for semantic errors (eg missing data, duplicate activity, ...)
	Err error
}

// Uploader supports uploading and status checking of an upload
type Uploader interface {
	// Upload uploads a file
	Upload(ctx context.Context, file *File) (Upload, error)
	// Status returns the processing status of a file
	Status(ctx context.Context, id UploadID) (Upload, error)
}

// File for uploading and exporting
type File struct {
	io.Reader `json:"-"`
	Filename  string `json:"filename,omitempty"`
	Name      string `json:"name"`
	Format    Format `json:"format"`
}

// Close the reader (if supported)
func (f *File) Close() error {
	if f.Reader == nil {
		return nil
	}
	if x, ok := f.Reader.(io.Closer); ok {
		return x.Close()
	}
	return nil
}

// Format of the file used in exporting and uploading
type Format int

const (
	// Original format (essentially a wildcard)
	FormatOriginal Format = iota // original
	// GPX format
	FormatGPX // gpx
	// TCX format
	FormatTCX // tcx
	// FIT format
	FormatFIT // fit
)

// MarshalJSON converts a Format enum to a string representation
func (f Format) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, f.String())), nil
}

// ToFormat converts a file extension (with or without the ".") to a Format
// If no predefined extension exists the Format Original is returned
func ToFormat(format string) Format {
	format = strings.ToLower(format)
	switch format {
	case ".gpx", "gpx":
		return FormatGPX
	case ".tcx", "tcx":
		return FormatTCX
	case ".fit", "fit":
		return FormatFIT
	default:
		return FormatOriginal
	}
}

// A PollerOption allows configuring the default poller
type PollerOption func(p *poller)

// WithInterval controls the duration between status polling
func WithInterval(interval time.Duration) PollerOption {
	return func(p *poller) {
		if interval > 0 {
			p.interval = interval
		}
	}
}

// WithIterations controls the max number of polling iterations
func WithIterations(iterations int) PollerOption {
	return func(p *poller) {
		if iterations > 0 {
			p.iterations = iterations
		}
	}
}

// Poller will continually check the status of an upload request
type Poller interface {
	// Poll the status of an upload
	//
	// The operation will continue until either it is completed, the context
	//  is canceled, or the maximum number of iterations have been exceeded.
	Poll(ctx context.Context, uploadID UploadID) <-chan *Poll
}

// NewPoller returns an instance of a Poller
func NewPoller(uploader Uploader, opts ...PollerOption) Poller {
	p := &poller{uploader: uploader, interval: pollInterval, iterations: pollIterations}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type poller struct {
	uploader   Uploader
	interval   time.Duration
	iterations int
}

func (p *poller) Poll(ctx context.Context, uploadID UploadID) <-chan *Poll {
	res := make(chan *Poll)
	go func() {
		defer close(res)
		for i := p.iterations; i > 0; i-- {
			upload, err := p.uploader.Status(ctx, uploadID)
			poll := &Poll{Upload: upload, Err: err}
			select {
			case <-ctx.Done():
				return
			case res <- poll:
				if poll.Err != nil || poll.Upload.Done() {
					return
				}
			}
			// wait for a bit to let the processing continue
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.interval):
			}
		}
		select {
		case <-ctx.Done():
			return
		case res <- &Poll{Err: ErrExceededIterations}:
		}
	}()
	return res
}
