package strava_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bzimmer/activity/strava"
)

func TestWebhookSubscribe(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(ack *strava.WebhookAcknowledgement, err error)
	}{
		{
			name: "webhook subscribe",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/push_subscriptions", func(w http.ResponseWriter, r *http.Request) {
					a.Equal(http.MethodPost, r.Method)
					a.Equal("someID", r.FormValue("client_id"))
					a.Equal("someSecret", r.FormValue("client_secret"))
					a.Equal("https://example.com/wh/callback", r.FormValue("callback_url"))
					a.Equal("verifyToken123", r.FormValue("verify_token"))
					http.ServeFile(w, r, "testdata/webhook_subscribe.json")
				})
			},
			after: func(ack *strava.WebhookAcknowledgement, err error) {
				a.NoError(err)
				a.NotNil(ack)
				a.Equal(int64(887228), ack.ID)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(
				tt.before, strava.WithClientCredentials("someID", "someSecret"))
			defer svr.Close()
			ack, err := client.Webhook.Subscribe(
				context.TODO(), "https://example.com/wh/callback", "verifyToken123")
			tt.after(ack, err)
		})
	}
}

func TestWebhookUnsubscribe(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(mux *http.ServeMux)
		after  func(err error)
	}{
		{
			name: "webhook unsubscribe",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/push_subscriptions/882722", func(w http.ResponseWriter, r *http.Request) {
					a.Equal(r.Method, http.MethodDelete)
					w.WriteHeader(http.StatusNoContent)
				})
			},
			after: func(err error) {
				a.NoError(err)
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(tt.before)
			defer svr.Close()
			err := client.Webhook.Unsubscribe(context.TODO(), 882722)
			tt.after(err)
		})
	}
}

func TestWebhookList(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	tests := []struct {
		name   string
		before func(*http.ServeMux)
		after  func([]*strava.WebhookSubscription, error)
	}{
		{
			name: "webhook list",
			before: func(mux *http.ServeMux) {
				mux.HandleFunc("/push_subscriptions", func(w http.ResponseWriter, r *http.Request) {
					a.Equal(r.Method, http.MethodGet)
					a.Equal("someID", r.FormValue("client_id"))
					a.Equal("someSecret", r.FormValue("client_secret"))
					http.ServeFile(w, r, "testdata/subscriptions.json")
				})
			},
			after: func(subs []*strava.WebhookSubscription, err error) {
				a.NoError(err)
				a.Equal(1, len(subs))
			},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, svr := newClientMust(
				tt.before, strava.WithClientCredentials("someID", "someSecret"))
			defer svr.Close()
			tt.after(client.Webhook.List(context.TODO()))
		})
	}
}

type TestSubscriber struct {
	verify    string
	challenge string
	msg       *strava.WebhookMessage
	fail      bool
}

func (t *TestSubscriber) SubscriptionRequest(challenge, verify string) error {
	t.verify = verify
	t.challenge = challenge
	if t.fail {
		return errors.New("failed")
	}
	return nil
}

func (t *TestSubscriber) MessageReceived(msg *strava.WebhookMessage) error {
	t.msg = msg
	if t.fail {
		return errors.New("failed")
	}
	return nil
}

func setupTestRouter() (*TestSubscriber, *http.ServeMux) {
	sub := &TestSubscriber{fail: false}
	mux := http.NewServeMux()
	mux.Handle("/webhook", strava.NewWebhookHandler(sub))
	return sub, mux
}

func TestWebhookEventHandler(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	sub, router := setupTestRouter()

	reader := strings.NewReader(`
	{
		"aspect_type": "update",
		"event_time": 1516126040,
		"object_id": 1360128428,
		"object_type": "activity",
		"owner_id": 18637089,
		"subscription_id": 120475,
		"updates": {
			"title": "Messy",
			"type": "Bike"

		}
	}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", reader)
	router.ServeHTTP(w, req)

	a.Equal(200, w.Code)
	a.NotNil(sub)
	a.Equal(18637089, sub.msg.OwnerID)
	a.Equal("Bike", sub.msg.Updates["type"])

	reader = strings.NewReader(`
	{
		"aspect_type": "update",
		"event_time": 1516126040,
		"object_id": 1360128428,
		"object_type": "activity",
		"owner_id": 18637089,
		"subscription_id": 120475,
		"updates": {
			"title": "Messy",
			"type": "Bike"

		}
	}`)

	sub.fail = true
	w = httptest.NewRecorder()
	req, _ = http.NewRequestWithContext(context.TODO(), http.MethodPost, "/webhook", reader)
	router.ServeHTTP(w, req)

	a.Equal(500, w.Code)
}

func TestWebhookSubscriptionHandler(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	sub, router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(
		context.TODO(), http.MethodGet, "/webhook?hub.verify_token=bar&hub.challenge=baz", nil)
	router.ServeHTTP(w, req)

	a.Equal(200, w.Code)
	a.NotNil(sub)
	a.Equal("bar", sub.verify)
	a.Equal("baz", sub.challenge)

	sub.fail = true
	w = httptest.NewRecorder()
	req, _ = http.NewRequestWithContext(
		context.TODO(), http.MethodGet, "/webhook?hub.verify_token=bar&hub.challenge=baz", nil)
	router.ServeHTTP(w, req)

	a.Equal(500, w.Code)
}
