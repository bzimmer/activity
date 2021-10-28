package cyclinganalytics_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bzimmer/activity/cyclinganalytics"
	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "query `me` user",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			a := assert.New(t)
			mux := http.NewServeMux()
			mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				enc := json.NewEncoder(w)
				a.NoError(enc.Encode(&cyclinganalytics.User{
					Email: "me@example.com",
					ID:    cyclinganalytics.UserID(1590343),
					Name:  "Some One",
				}))
			})

			svr := httptest.NewServer(mux)
			defer svr.Close()

			client, err := cyclinganalytics.NewClient(
				cyclinganalytics.WithBaseURL(svr.URL),
				cyclinganalytics.WithTokenCredentials("fooKey", "barToken", time.Time{}))
			a.NoError(err)
			me, err := client.User.Me(context.Background())
			a.NoError(err)
			a.NotNil(me)
			a.Equal("me@example.com", me.Email)
		})
	}
}
