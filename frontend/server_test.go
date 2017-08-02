package main

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct {
}

func (d *fakeDB) GetLessons(ctx context.Context, cursor string) (map[string]entry, string, error) {
	return map[string]entry{"key1": entry{}}, "next cursor", nil
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
