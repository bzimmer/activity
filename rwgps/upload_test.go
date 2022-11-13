package rwgps_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUploader(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	client, svr := newClient(nil)
	defer svr.Close()
	a.NotNil(client)
	uploader := client.Uploader()
	a.NotNil(uploader)
}
