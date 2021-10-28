package cyclinganalytics_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/bzimmer/httpwares"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/bzimmer/activity/cyclinganalytics"
)

func newClient(status int, filename string) (*cyclinganalytics.Client, error) {
	return cyclinganalytics.NewClient(
		cyclinganalytics.WithTransport(&httpwares.TestDataTransport{
			Status:      status,
			Filename:    filename,
			ContentType: "application/json"}),
		cyclinganalytics.WithTokenCredentials("fooKey", "barToken", time.Time{}))
}

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
