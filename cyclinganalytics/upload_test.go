package cyclinganalytics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bzimmer/activity/cyclinganalytics"
	"github.com/stretchr/testify/assert"
)

func TestUpload(t *testing.T) {
	tests := []struct {
		name, err string
	}{
		{
			name: "uploader error",
			err:  "missing upload file",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			a := assert.New(t)
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			})

			svr := httptest.NewServer(mux)
			defer svr.Close()

			client, err := cyclinganalytics.NewClient(
				cyclinganalytics.WithBaseURL(svr.URL),
				cyclinganalytics.WithTokenCredentials("fooKey", "barToken", time.Time{}))
			a.NoError(err)
			uploader := client.Uploader()
			a.NotNil(uploader)
			upload, err := uploader.Upload(context.Background(), nil)
			if tt.err != "" {
				a.Error(err)
				a.Nil(upload)
				a.Contains(err.Error(), tt.err)
				return
			}
		})
	}
}
