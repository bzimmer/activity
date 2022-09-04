package zwift_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRefresh(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		n, err := w.Write([]byte(`{
			"access_token":"000aaabbbccc999",
			"token_type":"bearer",
			"expires_in":3600,
			"refresh_token":"TotalNonsense",
			"scope":"user"
		  }`))
		a.Greater(n, 0)
		a.NoError(err)
	})

	tests := []struct {
		name string
	}{
		{
			name: "success",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClient(t, mux)
			defer svr.Close()
			token, err := client.Auth.Refresh(context.Background(), "foo", "bar")
			a.NoError(err)
			a.NotNil(token)
			a.Equal("TotalNonsense", token.RefreshToken)
		})
	}
}
