package zwift_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/zwift"
)

func TestExporter(t *testing.T) {
	a := assert.New(t)
	client, err := zwift.NewClient()
	a.NoError(err)
	a.NotNil(client)
	a.NotNil(client.Exporter())
}

func TestActivity(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles/1037/activities/882920", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Activity{ID: 882920}))
	})

	tests := []struct {
		name              string
		athlete, activity int64
		err               string
	}{
		{
			name:     "success",
			athlete:  1037,
			activity: 882920,
		},
		{
			name:     "failure",
			athlete:  1099,
			activity: 882920,
			err:      "Not Found",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, mux)
			defer svr.Close()
			activity, err := client.Activity.Activity(context.Background(), tt.athlete, tt.activity)
			switch {
			case tt.err != "":
				a.Error(err)
				a.Nil(activity)
			default:
				a.NoError(err)
				a.NotNil(activity)
				a.Equal(tt.activity, activity.ID)
			}
		})
	}
}

func TestActivities(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles/1037/activities/", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		var res []*zwift.Activity
		for i := 0; i < 5; i++ {
			res = append(res, &zwift.Activity{ID: 882920 + int64(i)})
		}
		a.NoError(enc.Encode(res))
	})

	tests := []struct {
		name              string
		athlete, activity int64
		err               string
	}{
		{
			name:     "success",
			athlete:  1037,
			activity: 882920,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, mux, zwift.WithHTTPTracing(true))
			defer svr.Close()
			activities, err := client.Activity.Activities(
				context.Background(), tt.athlete, activity.Pagination{Start: 1, Total: 5})
			switch {
			case tt.err != "":
				a.Error(err)
				a.Nil(activities)
			default:
				a.NoError(err)
				a.NotNil(activities)
				a.Len(activities, 5)
			}
		})
	}
}
