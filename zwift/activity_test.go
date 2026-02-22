package zwift_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	mux.HandleFunc("/api/profiles/1037/activities/882920", func(w http.ResponseWriter, _ *http.Request) {
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
	mux.HandleFunc("/api/profiles/1037/activities/", func(w http.ResponseWriter, _ *http.Request) {
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

func TestExportActivity(t *testing.T) {
	t.Parallel()

	// Mock FIT file content
	mockFitData := []byte("MOCK_FIT_FILE_CONTENT")

	tests := []struct {
		name           string
		activity       *zwift.Activity
		mockResponse   func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errMsg         string
		validateResult func(t *testing.T, export *activity.Export)
	}{
		{
			name: "successful export",
			activity: &zwift.Activity{
				ID:            12345,
				FitFileBucket: "zwift-activity-prod",
				FitFileKey:    "2024/01/activity.fit",
			},
			mockResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Disposition", "filename=2024-01-15.fit")
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(mockFitData)
			},
			wantErr: false,
			validateResult: func(t *testing.T, export *activity.Export) {
				a := assert.New(t)
				r := require.New(t)
				r.NotNil(export)
				a.Equal(int64(12345), export.ID)
				a.Equal("2024-01-15.fit", export.Name)
				a.Equal(activity.FormatFIT, export.Format)

				// Read and verify content
				buf := &bytes.Buffer{}
				_, err := io.Copy(buf, export.Reader)
				a.NoError(err)
				a.Equal(mockFitData, buf.Bytes())
			},
		},
		{
			name: "activity not found - 404",
			activity: &zwift.Activity{
				ID:            99999,
				FitFileBucket: "zwift-activity-prod",
				FitFileKey:    "nonexistent.fit",
			},
			mockResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
			errMsg:  "activity not found",
		},
		{
			name: "server error - 500",
			activity: &zwift.Activity{
				ID:            12345,
				FitFileBucket: "zwift-activity-prod",
				FitFileKey:    "error.fit",
			},
			mockResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
			errMsg:  "error code: 500",
		},
		{
			name: "forbidden - 403",
			activity: &zwift.Activity{
				ID:            12345,
				FitFileBucket: "zwift-activity-prod",
				FitFileKey:    "forbidden.fit",
			},
			mockResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr: true,
			errMsg:  "error code: 403",
		},
		{
			name: "invalid bucket name - validation fails",
			activity: &zwift.Activity{
				ID:            12345,
				FitFileBucket: "malicious-bucket",
				FitFileKey:    "activity.fit",
			},
			mockResponse: func(_ http.ResponseWriter, _ *http.Request) {
				// Should not be called due to validation failure
				t.Error("mockResponse should not be called for invalid bucket")
			},
			wantErr: true,
			errMsg:  "invalid bucket: expected Zwift bucket",
		},
		{
			name: "export with path in key",
			activity: &zwift.Activity{
				ID:            67890,
				FitFileBucket: "zwift-exports",
				FitFileKey:    "user/123/activities/2024-01-15-ride.fit",
			},
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify the URL path
				expectedPath := "/user/123/activities/2024-01-15-ride.fit"
				if !strings.Contains(r.URL.Path, expectedPath) {
					t.Errorf("Expected path to contain %s, got %s", expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Disposition", "filename=2024-01-15-ride.fit")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(mockFitData)
			},
			wantErr: false,
			validateResult: func(t *testing.T, export *activity.Export) {
				a := assert.New(t)
				a.Equal("2024-01-15-ride.fit", export.Name)
			},
		},
		{
			name: "malformed content-disposition header",
			activity: &zwift.Activity{
				ID:            12345,
				FitFileBucket: "zwift-activity-prod",
				FitFileKey:    "activity.fit",
			},
			mockResponse: func(w http.ResponseWriter, _ *http.Request) {
				// Invalid Content-Disposition format
				w.Header().Set("Content-Disposition", "invalid disposition header")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(mockFitData)
			},
			wantErr: true,
			errMsg:  "mime:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			// Create a test server to mock S3 responses
			s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify it's a GET request
				a.Equal(http.MethodGet, r.Method)

				// Verify the URL path matches the FitFileKey
				a.Equal("/"+tt.activity.FitFileKey, r.URL.Path)

				tt.mockResponse(w, r)
			}))
			defer s3Server.Close()

			// Create a custom HTTP client that redirects S3 requests to our test server
			customTransport := &mockS3Transport{
				testServer: s3Server,
			}

			// Create a test client using newClient helper with custom transport
			mux := http.NewServeMux()
			client, svr := newClient(t, mux, zwift.WithHTTPClient(&http.Client{
				Transport: customTransport,
			}))
			defer svr.Close()

			// Execute the test
			ctx := context.Background()
			export, err := client.Activity.ExportActivity(ctx, tt.activity)

			if tt.wantErr {
				a.Error(err)
				if tt.errMsg != "" {
					a.Contains(err.Error(), tt.errMsg)
				}
				a.Nil(export)
			} else {
				a.NoError(err)
				if tt.validateResult != nil {
					tt.validateResult(t, export)
				}
			}
		})
	}
}

// mockS3Transport is a custom RoundTripper that redirects S3 requests to a test server
type mockS3Transport struct {
	testServer *httptest.Server
}

func (m *mockS3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect S3 requests to test server
	if strings.Contains(req.URL.Host, ".s3.amazonaws.com") {
		// Replace the host with test server host
		newURL := *req.URL
		newURL.Scheme = "http"
		newURL.Host = strings.TrimPrefix(m.testServer.URL, "http://")

		// Create new request with test server URL
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL.String(), req.Body)
		if err != nil {
			return nil, err
		}

		// Copy headers
		newReq.Header = req.Header

		// Use default transport for the test server request
		return http.DefaultTransport.RoundTrip(newReq)
	}

	// For non-S3 requests, use default transport
	return http.DefaultTransport.RoundTrip(req)
}

func TestExportActivityContextCancellation(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// Create a custom HTTP client that simulates a slow response
	slowTransport := &slowTransport{}

	mux := http.NewServeMux()
	client, svr := newClient(t, mux, zwift.WithHTTPClient(&http.Client{
		Transport: slowTransport,
	}))
	defer svr.Close()

	act := &zwift.Activity{
		ID:            12345,
		FitFileBucket: "zwift-activity-prod",
		FitFileKey:    "activity.fit",
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	export, err := client.Activity.ExportActivity(ctx, act)
	a.Error(err)
	a.Nil(export)
	a.Equal(context.Canceled, err)
}

// slowTransport simulates a slow/hanging request
type slowTransport struct{}

func (s *slowTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if context is already done
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	default:
	}

	// Simulate slow response by blocking until context is done
	<-req.Context().Done()
	return nil, req.Context().Err()
}
