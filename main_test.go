package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostMetrics(t *testing.T) {
	assert := assert.New(t)

	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		expectedBody := `go_gc_duration_seconds{quantile="0"} 3.4204e-05`
		assert.Contains(string(body), expectedBody)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`ok`))
		require.NoError(t, err)
	}))

	plugin.URL = apiStub.URL
	m := "# TYPE go_gc_duration_seconds summary\ngo_gc_duration_seconds{quantile=\"0\"} 3.4204e-05\n"
	err := postMetrics("foo", "bar", m)
	assert.NoError(err)
}

func TestMain(t *testing.T) {
	assert := assert.New(t)
	file, _ := ioutil.TempFile(os.TempDir(), "sensu-prometheus-pushgateway-handler-")
	defer func() {
		_ = os.Remove(file.Name())
	}()

	event := corev2.FixtureEvent("entity1", "check1")
	event.Check = nil
	event.Metrics = corev2.FixtureMetrics()
	eventJSON, _ := json.Marshal(event)
	_, err := file.WriteString(string(eventJSON))
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	_, err = file.Seek(0, 0)
	require.NoError(t, err)
	os.Stdin = file
	requestReceived := false

	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`ok`))
		require.NoError(t, err)
	}))

	oldArgs := os.Args
	os.Args = []string{"sensu-prometheus-pushgateway-handler", "-u", apiStub.URL, "-j", "foo", "-i", "bar"}
	defer func() { os.Args = oldArgs }()

	main()
	assert.True(requestReceived)
}
