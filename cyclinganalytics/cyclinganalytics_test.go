package cyclinganalytics_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/cyclinganalytics"
)

func TestWith(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, err := cyclinganalytics.NewClient(
		cyclinganalytics.WithConfig(oauth2.Config{}),
		cyclinganalytics.WithHTTPTracing(true),
		cyclinganalytics.WithHTTPClient(http.DefaultClient),
		cyclinganalytics.WithToken(&oauth2.Token{}),
		cyclinganalytics.WithAutoRefresh(context.Background()),
		cyclinganalytics.WithRateLimiter(rate.NewLimiter(rate.Every(time.Second), 10)),
		cyclinganalytics.WithClientCredentials("foo", "bar"))
	a.NoError(err)
	a.NotNil(client)
}

func TestWithTransport(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name      string
		transport http.RoundTripper
		wantErr   bool
	}{
		{
			name:      "valid transport",
			transport: http.DefaultTransport,
			wantErr:   false,
		},
		{
			name:      "nil transport returns error",
			transport: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := cyclinganalytics.NewClient(cyclinganalytics.WithTransport(tt.transport))
			if tt.wantErr {
				a.Error(err)
				a.Nil(client)
			} else {
				a.NoError(err)
				a.NotNil(client)
			}
		})
	}
}

func TestStreamSets(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	client, err := cyclinganalytics.NewClient()
	a.NoError(err)
	a.NotNil(client)

	sets := client.Rides.StreamSets()
	a.NotEmpty(sets)
	a.Contains(sets, "power")
	a.Contains(sets, "heartrate")
	a.Contains(sets, "distance")
}
