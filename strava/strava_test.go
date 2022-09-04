package strava_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"time"

	"github.com/bzimmer/activity/strava"
)

func newClient(before func(*http.ServeMux), opts ...strava.Option) (*strava.Client, *httptest.Server) {
	mux := http.NewServeMux()
	before(mux)
	svr := httptest.NewServer(mux)
	options := []strava.Option{
		strava.WithBaseURL(svr.URL),
		strava.WithHTTPTracing(false),
		strava.WithTokenCredentials("key", "token", time.Time{}),
	}
	client, err := strava.NewClient(append(options, opts...)...)
	if err != nil {
		panic(err)
	}
	return client, svr
}

type ManyHandler struct {
	Filename string
	Total    int
	total    int
}

func (m *ManyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	n, err := strconv.Atoi(q.Get("per_page"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if m.Total > 0 {
		n = m.Total
		if m.total >= m.Total {
			n = 0
		}
	}
	m.total += n

	w.WriteHeader(http.StatusOK)
	data, err := os.ReadFile(m.Filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write([]byte("[")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for i := 0; i < n; i++ {
		if _, err = w.Write(data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if i+1 < n {
			if _, err = w.Write([]byte(",")); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	if _, err = w.Write([]byte("]")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
