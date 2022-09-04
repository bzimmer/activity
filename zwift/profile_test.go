package zwift_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/zwift"
)

func TestProfile(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles/abcxyz", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Profile{FirstName: "barney"}))
	})
	mux.HandleFunc("/api/profiles/me", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		a.NoError(enc.Encode(&zwift.Profile{FirstName: "betty"}))
	})

	tests := []struct {
		name      string
		user      string
		firstname string
	}{
		{
			name:      "success me",
			user:      "",
			firstname: "betty",
		},
		{
			name:      "success",
			user:      "abcxyz",
			firstname: "barney",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, mux)
			defer svr.Close()
			profile, err := client.Profile.Profile(context.Background(), tt.user)
			a.NoError(err)
			a.NotNil(profile)
			a.Equal(tt.firstname, profile.FirstName)
		})
	}
}
