package health

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	var tests = []struct {
		msg        string
		jsonResp   string
		wantStatus int
	}{
		{
			msg:        "status green",
			jsonResp:   `{"status":"green"}`,
			wantStatus: 200,
		}, {
			msg:        "status yellow",
			jsonResp:   `{"status":"yellow"}`,
			wantStatus: 200,
		}, {
			msg:        "garbage json",
			jsonResp:   `garbaaaaage`,
			wantStatus: 500,
		},
	}
	for _, test := range tests {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, test.jsonResp)
		}))
		defer ts.Close()
		h := NewElasticHealthChecker(ts.URL)
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		resp := w.Result()
		if got, want := resp.StatusCode, test.wantStatus; got != want {
			t.Errorf("[%s] resp status code got=%d, want=%d", test.msg, got, want)
		}
	}
}
