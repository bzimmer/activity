package zwift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateZwiftS3URL(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid zwift S3 URL",
			url:     "https://zwift-activity-prod.s3.amazonaws.com/activity.fit",
			wantErr: false,
		},
		{
			name:    "valid zwift S3 URL with path",
			url:     "https://zwift-exports.s3.amazonaws.com/path/to/file.fit",
			wantErr: false,
		},
		{
			name:    "valid zwift S3 URL - case insensitive bucket",
			url:     "https://ZWIFT-data.s3.amazonaws.com/file.fit",
			wantErr: false,
		},
		{
			name:    "valid zwift S3 URL - zwift in middle of bucket name",
			url:     "https://prod-zwift-data.s3.amazonaws.com/file.fit",
			wantErr: false,
		},
		{
			name:    "invalid URL - malformed",
			url:     "://invalid-url",
			wantErr: true,
			errMsg:  "invalid URL",
		},
		{
			name:    "invalid URL - http scheme",
			url:     "http://zwift-activity.s3.amazonaws.com/file.fit",
			wantErr: true,
			errMsg:  "invalid URL scheme: expected https, got http",
		},
		{
			name:    "invalid URL - ftp scheme",
			url:     "ftp://zwift-activity.s3.amazonaws.com/file.fit",
			wantErr: true,
			errMsg:  "invalid URL scheme: expected https, got ftp",
		},
		{
			name:    "invalid URL - no scheme",
			url:     "zwift-activity.s3.amazonaws.com/file.fit",
			wantErr: true,
			errMsg:  "invalid URL scheme: expected https, got",
		},
		{
			name:    "invalid host - not S3",
			url:     "https://zwift.com/file.fit",
			wantErr: true,
			errMsg:  "invalid host: expected *.s3.amazonaws.com, got zwift.com",
		},
		{
			name:    "invalid host - not amazonaws",
			url:     "https://zwift-bucket.s3.google.com/file.fit",
			wantErr: true,
			errMsg:  "invalid host: expected *.s3.amazonaws.com, got zwift-bucket.s3.google.com",
		},
		{
			name:    "invalid host - missing bucket name",
			url:     "https://s3.amazonaws.com/file.fit",
			wantErr: true,
			errMsg:  "invalid host: expected *.s3.amazonaws.com, got s3.amazonaws.com",
		},
		{
			name:    "invalid bucket - no zwift in name",
			url:     "https://activity-prod.s3.amazonaws.com/file.fit",
			wantErr: true,
			errMsg:  "invalid bucket: expected Zwift bucket, got activity-prod",
		},
		{
			name:    "invalid bucket - wrong service name",
			url:     "https://strava-exports.s3.amazonaws.com/file.fit",
			wantErr: true,
			errMsg:  "invalid bucket: expected Zwift bucket, got strava-exports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateZwiftS3URL(tt.url)
			if tt.wantErr {
				a.Error(err)
				if tt.errMsg != "" {
					a.Contains(err.Error(), tt.errMsg)
				}
			} else {
				a.NoError(err)
			}
		})
	}
}
