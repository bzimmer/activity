package zwift_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/zwift"
)

func TestModel(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

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
			src := zwift.Activity{
				StartDate: zwift.Datetime{Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
			}
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			a.NoError(enc.Encode(src))
			var dst zwift.Activity
			dec := json.NewDecoder(&buf)
			a.NoError(dec.Decode(&dst))
			a.Equal(src.StartDate, dst.StartDate)
		})
	}
}
