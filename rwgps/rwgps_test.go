package rwgps_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bzimmer/activity/rwgps"
	"github.com/bzimmer/httpwares"
)

func newClient(status int, filename string) (*rwgps.Client, error) {
	return rwgps.NewClient(
		rwgps.WithTransport(&httpwares.TestDataTransport{
			Status:      status,
			Filename:    filename,
			ContentType: "application/json",
			Requester: func(req *http.Request) error {
				var body map[string]interface{}
				decoder := json.NewDecoder(req.Body)
				if err := decoder.Decode(&body); err != nil {
					return err
				}
				// confirm the body has the expected key:value pairs
				for key, value := range map[string]string{
					"apikey":     "fooKey",
					"version":    "2",
					"auth_token": "barToken",
				} {
					v := body[key]
					if v != value {
						return fmt.Errorf("expected %s == '%v', not '%v'", key, value, v)
					}
				}
				return nil
			},
		}),
		rwgps.WithClientCredentials("fooKey", ""),
		rwgps.WithTokenCredentials("barToken", "", time.Time{}),
		rwgps.WithHTTPTracing(false),
	)
}
