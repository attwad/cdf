package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/attwad/cdf/data"
)

type fakeDB struct {
}

func (d *fakeDB) GetLessons(ctx context.Context, cursor string) (map[string]data.Entry, string, error) {
	return map[string]data.Entry{"key1": data.Entry{}}, "next cursor", nil
}

func TestIndex(t *testing.T) {
	db := &fakeDB{}
	s := &server{ctx: context.Background(), db: db}
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.ServeIndex(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if !strings.Contains(string(body), "mdl-card") {
		t.Errorf("Expected body to contain an mdl-card but did not: %s", string(body))
	}
	if !strings.Contains(string(body), "next%20cursor") {
		t.Errorf("Expected next cursor link but was not found: %s", string(body))
	}
}

func TestSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer ts.Close()

	db := &fakeDB{}
	s := &server{
		ctx: context.Background(),
		httpClient: &http.Client{
			Timeout: time.Second * 2,
		},
		db:             db,
		elasticAddress: ts.URL,
	}
	req := httptest.NewRequest("GET", "/search?q=myquery", nil)
	w := httptest.NewRecorder()
	s.ServeSearch(w, req)

	// TODO: Check resp.
}
