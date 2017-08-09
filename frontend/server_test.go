package main

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attwad/cdf/data"
)

type fakeDB struct {
}

func (d *fakeDB) GetLessons(ctx context.Context, cursor string) ([]data.Entry, string, error) {
	return []data.Entry{data.Entry{}}, "next cursor", nil
}

func TestAPIServeLessons(t *testing.T) {
	db := &fakeDB{}
	s := &server{ctx: context.Background(), db: db}
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.APIServeLessons(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if !strings.Contains(string(body), "title\":") {
		t.Errorf("Expected title but was not found: %s", string(body))
	}

	if !strings.Contains(string(body), "next cursor") {
		t.Errorf("Expected next cursor link but was not found: %s", string(body))
	}
}
