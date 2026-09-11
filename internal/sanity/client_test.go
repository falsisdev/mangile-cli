package sanity

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type recordingTransport struct {
	lastPath string
	lastBody string
	status   int
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.lastPath = req.URL.String()
	body, _ := io.ReadAll(req.Body)
	t.lastBody = string(body)
	req.Body.Close()
	return &http.Response{
		StatusCode: t.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"result":[]}`)),
		Request:    req,
	}, nil
}

func TestQueryPathParams(t *testing.T) {
	rt := &recordingTransport{status: http.StatusOK}
	c := New("tok", "proj", "production", "v2024-01-01")
	c.http = &http.Client{Transport: rt}

	var res []map[string]any
	if err := c.Query(context.Background(), `*[_type == $type]`, map[string]any{"type": "manga"}, &res); err != nil {
		t.Fatalf("Query hatası: %v", err)
	}
	if strings.Contains(rt.lastPath, "returnQueryMetadata") {
		t.Fatalf("geçersiz parametre istekte görünüyor: %s", rt.lastPath)
	}
	if !strings.Contains(rt.lastPath, "perspective=raw") {
		t.Fatalf("perspective=raw eksik: %s", rt.lastPath)
	}
	if !strings.Contains(rt.lastPath, "/data/query/production") {
		t.Fatalf("beklenmeyen yol: %s", rt.lastPath)
	}
	if !strings.Contains(rt.lastBody, `"type":"manga"`) {
		t.Fatalf("parametreler gönderilmedi: %s", rt.lastBody)
	}
}

func TestQueryErrorPropagated(t *testing.T) {
	rt := &recordingTransport{status: http.StatusBadRequest}
	c := New("tok", "proj", "production", "v2024-01-01")
	c.http = &http.Client{Transport: rt}

	var res []map[string]any
	err := c.Query(context.Background(), `*[_type == $type]`, map[string]any{"type": "manga"}, &res)
	if err == nil {
		t.Fatal("hata bekleniyordu, nil döndü")
	}
	if !strings.Contains(err.Error(), "sorgu hatası") {
		t.Fatalf("beklenmeyen hata mesajı: %v", err)
	}
}
