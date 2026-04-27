package rwgps_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/rwgps"
)

func TestUploader(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, svr := newClient(nil)
	defer svr.Close()
	a.NotNil(client)
	uploader := client.Uploader()
	a.NotNil(uploader)
}

func TestUploaderUploadAndStatus(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, svr := newClient(func(mux *http.ServeMux) {
		mux.HandleFunc("/trips.json", func(w http.ResponseWriter, _ *http.Request) {
			enc := json.NewEncoder(w)
			_ = enc.Encode(&rwgps.Upload{TaskID: 1234, Success: 0})
		})
		mux.HandleFunc("/queued_tasks/status.json", func(w http.ResponseWriter, _ *http.Request) {
			enc := json.NewEncoder(w)
			_ = enc.Encode(&rwgps.Upload{
				TaskID:  1234,
				Success: 1,
			})
		})
	})
	defer svr.Close()

	uploader := client.Uploader()
	ctx := context.TODO()

	file := &activity.File{
		Reader: strings.NewReader("<gpx/>"),
		Name:   "ride.gpx",
		Format: activity.FormatGPX,
	}

	upload, err := uploader.Upload(ctx, file)
	a.NoError(err)
	a.NotNil(upload)
	a.Equal(activity.UploadID(1234), upload.Identifier())

	status, err := uploader.Status(ctx, upload.Identifier())
	a.NoError(err)
	a.NotNil(status)
	a.True(status.Done())
}
