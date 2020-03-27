package phpdaemon

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netdata/go-orchestrator/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testURL = "http://127.0.0.1:38001"
)

var testFullStatusData, _ = ioutil.ReadFile("testdata/fullstatus.json")

func Test_testData(t *testing.T) {
	assert.NotEmpty(t, testFullStatusData)
}

func TestNew(t *testing.T) {
	job := New()

	assert.Implements(t, (*module.Module)(nil), job)
	assert.Equal(t, defaultURL, job.UserURL)
	assert.Equal(t, defaultHTTPTimeout, job.Timeout.Duration)
}

func TestPHPDaemon_Init(t *testing.T) {
	job := New()

	require.True(t, job.Init())
	assert.NotNil(t, job.client)
}

func TestPHPDaemon_Check(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(testFullStatusData)
			}))
	defer ts.Close()

	job := New()
	job.UserURL = ts.URL
	require.True(t, job.Init())
	assert.True(t, job.Check())
}

func TestPHPDaemon_CheckNG(t *testing.T) {
	job := New()
	job.UserURL = testURL
	require.True(t, job.Init())
	assert.False(t, job.Check())
}

func TestPHPDaemon_Charts(t *testing.T) {
	job := New()

	assert.NotNil(t, job.Charts())
	assert.False(t, job.charts.Has(uptimeChart.ID))

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(testFullStatusData)
			}))
	defer ts.Close()

	job.UserURL = ts.URL
	require.True(t, job.Init())
	assert.True(t, job.Check())
	assert.True(t, job.charts.Has(uptimeChart.ID))
}

func TestPHPDaemon_Cleanup(t *testing.T) {
	assert.NotPanics(t, New().Cleanup)
}

func TestPHPDaemon_Collect(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(testFullStatusData)
			}))
	defer ts.Close()

	job := New()
	job.UserURL = ts.URL
	require.True(t, job.Init())
	assert.True(t, job.Check())

	expected := map[string]int64{
		"alive":       350,
		"busy":        200,
		"idle":        50,
		"init":        20,
		"initialized": 10,
		"preinit":     20,
		"reloading":   100,
		"shutdown":    500,
		"uptime":      15765,
	}

	assert.Equal(t, expected, job.Collect())

}

func TestPHPDaemon_InvalidData(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("hello and goodbye"))
			}))
	defer ts.Close()

	job := New()
	job.UserURL = ts.URL
	require.True(t, job.Init())
	assert.False(t, job.Check())
}

func TestPHPDaemon_404(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
	defer ts.Close()

	job := New()
	job.UserURL = ts.URL
	require.True(t, job.Init())
	assert.False(t, job.Check())
}
