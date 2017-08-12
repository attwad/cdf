package main

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attwad/cdf/data"
	"github.com/attwad/cdf/frontend/db"
	"github.com/attwad/cdf/frontend/search"
)

type fakeDB struct {
	filter db.Filter
}

func (d *fakeDB) GetLessons(ctx context.Context, cursor string, filter db.Filter) ([]data.Entry, string, error) {
	d.filter = filter
	return []data.Entry{data.Entry{}}, "next cursor", nil
}

type fakeSearcher struct{}

func (fs *fakeSearcher) Search(string) (*search.Response, error) {
	return &search.Response{TookMs: 42}, nil
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

func TestAPIServeLessonsFilterConverted(t *testing.T) {
	fdb := &fakeDB{}
	s := &server{ctx: context.Background(), db: fdb}
	req := httptest.NewRequest("GET", "/?filter=converted", nil)
	w := httptest.NewRecorder()
	s.APIServeLessons(w, req)

	if got, want := fdb.filter, db.FilterOnlyConverted; got != want {
		t.Errorf("filter got=%v, want=%v", got, want)
	}
}

func TestAPISearch(t *testing.T) {
	fs := &fakeSearcher{}
	s := &server{ctx: context.Background(), searcher: fs}
	req := httptest.NewRequest("GET", "/search?q=myquery", nil)
	w := httptest.NewRecorder()
	s.APIServeSearch(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	if !strings.Contains(string(body), "myquery") {
		t.Errorf("Expected %q to be in the output but was not found: %s", "myquery", string(body))
	}
	if !strings.Contains(string(body), "42") {
		t.Errorf("Expected %d to be in the output but was not found: %s", 42, string(body))
	}
}
