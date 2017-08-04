package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/attwad/cdf/data"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"
)

const pageSize = 50

var (
	hostPort  = flag.String("listen_addr", "127.0.0.1:8080", "Address to listen on.")
	projectID = flag.String("project_id", "college-de-france", "Google cloud project.")
	tmplPath  = flag.String("template_path", "", "Path to the templates directory")
	tmpl      = template.Must(template.ParseGlob(*tmplPath + "*.html"))
)

type indexPage struct {
	Cursor  string                `json:"cursor"`
	Entries map[string]data.Entry `json:"entries"`
}

type server struct {
	ctx            context.Context
	db             dbWrapper
	httpClient     *http.Client
	elasticAddress string
}

type dbWrapper interface {
	GetLessons(ctx context.Context, cursorStr string) (map[string]data.Entry, string, error)
}

type datastoreWrapper struct {
	client *datastore.Client
}

func (d *datastoreWrapper) GetLessons(ctx context.Context, cursorStr string) (map[string]data.Entry, string, error) {
	lessons := make(map[string]data.Entry, 0)
	query := datastore.NewQuery("Entry").Order("-Scraped").Limit(50)
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return nil, "", fmt.Errorf("bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
	}
	var e data.Entry
	it := d.client.Run(ctx, query)
	for {
		key, err := it.Next(&e)
		for err == iterator.Done {
			nextCursor, err := it.Cursor()
			if err != nil {
				return nil, "", fmt.Errorf("getting next cursor: %v", err)
			}
			return lessons, nextCursor.String(), nil
		}
		if err != nil {
			return nil, "", fmt.Errorf("failed fetching results: %v", err)
		}
		lessons[key.Encode()] = e
	}
}

func (s *server) ServeIndex(w http.ResponseWriter, r *http.Request) {
	lessons, cursor, err := s.db.GetLessons(s.ctx, r.URL.Query().Get("cursor"))
	if err != nil {
		log.Println("Could not read lessons from db:", err)
		http.Error(w, "Could not read lessons from DB", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "index.html", &indexPage{
		Entries: lessons,
		Cursor:  cursor,
	}); err != nil {
		http.Error(w, "Could not write template", http.StatusInternalServerError)
		return
	}
}

func (s *server) ServeSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		http.Error(w, "empty query", http.StatusBadRequest)
		return
	}
	u, err := url.Parse(s.elasticAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Path = "_search"
	type simpleQueryString struct {
		Query           string   `json:"query"`
		Analyzer        string   `json:"analyzer"`
		Fields          []string `json:"fields"`
		DefaultOperator string   `json:"default_operator"`
	}
	type searchQuery struct {
		SimpleQueryString simpleQueryString `json:"simple_query_string"`
	}
	type searchRequest struct {
		Query searchQuery `json:"query"`
	}
	body := &searchRequest{
		Query: searchQuery{
			SimpleQueryString: simpleQueryString{
				Query:           q,
				Analyzer:        "french",
				Fields:          []string{"transcript"},
				DefaultOperator: "and",
			},
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req, err := http.NewRequest("GET", u.String(), bytes.NewReader(jsonBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	type source struct {
		Title      string `json:"title"`
		Lecturer   string `json:"lecturer"`
		Chaire     string `json:"chaire"`
		Type       string `json:"type"`
		Transcript string `json:"transcript"`
	}
	type hit struct {
		Source source `json:"_source"`
	}
	type hits struct {
		Total int   `json:"total"`
		Hits  []hit `json:"hits"`
	}
	type searchResponse struct {
		TookMs   int  `json:"took"`
		TimedOut bool `json:"timed_out"`
		Hits     hits
	}
	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "search.html", &sr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		ctx: ctx,
		db:  &datastoreWrapper{client: client},
		httpClient: &http.Client{
			Timeout: time.Second * 2,
		},
		elasticAddress: "http://127.0.0.1:9200",
	}
	http.HandleFunc("/", s.ServeIndex)
	http.HandleFunc("/search", s.ServeSearch)

	log.Fatal(http.ListenAndServe(*hostPort, nil))
}
