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
	"os"
	"strings"
	"time"

	"github.com/attwad/cdf/data"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"
)

const pageSize = 15

var (
	hostPort  = flag.String("listen_addr", "127.0.0.1:8080", "Address to listen on.")
	projectID = flag.String("project_id", "college-de-france", "Google cloud project.")
	tmplPath  = flag.String("template_path", "", "Path to the templates directory")
	tmpl      = template.Must(template.ParseGlob(*tmplPath + "*.html"))
)

type indexPage struct {
	Query  string
	Cursor string
	// TODO: map iteration is non deterministic, feels weird on page reloads...
	Entries map[string]data.Entry
}

type server struct {
	ctx      context.Context
	db       dbWrapper
	searcher Searcher
}

type source struct {
	Title      string `json:"title"`
	Lecturer   string `json:"lecturer"`
	Chaire     string `json:"chaire"`
	Type       string `json:"type"`
	Language   string `json:"lang"`
	URL        string `json:"source_url"`
	Transcript string `json:"transcript"`
}
type hit struct {
	Source source `json:"_source"`
}
type hits struct {
	Total int   `json:"total"`
	Hits  []hit `json:"hits"`
}
type JsonSearchResponse struct {
	TookMs   int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     hits
}

type Searcher interface {
	Search(string) (*JsonSearchResponse, error)
}

type elasticSearcher struct {
	httpClient     *http.Client
	elasticAddress string
}

func (e *elasticSearcher) Search(q string) (*JsonSearchResponse, error) {
	u, err := url.Parse(e.elasticAddress)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	req, err := http.NewRequest("GET", u.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var jsr JsonSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsr); err != nil {
		return nil, err
	}
	return &jsr, nil
}

type dbWrapper interface {
	GetLessons(ctx context.Context, cursorStr string) (map[string]data.Entry, string, error)
}

type datastoreWrapper struct {
	client *datastore.Client
}

func (d *datastoreWrapper) GetLessons(ctx context.Context, cursorStr string) (map[string]data.Entry, string, error) {
	lessons := make(map[string]data.Entry, 0)
	query := datastore.NewQuery("Entry").Order("-Scraped").Limit(pageSize)
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return nil, "", fmt.Errorf("bad cursor %q: %v", cursorStr, err)
		}
		query = query.Start(cursor)
		log.Println("with cursor")
	} else {
		log.Println("No cursor")
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
		Query:   "",
		Entries: lessons,
		Cursor:  cursor,
	}); err != nil {
		http.Error(w, "Could not write template", http.StatusInternalServerError)
		return
	}
}

func (s *server) APIServeLessons(w http.ResponseWriter, r *http.Request) {
	lessons, cursor, err := s.db.GetLessons(s.ctx, r.URL.Query().Get("cursor"))
	if err != nil {
		log.Println("Could not read lessons from db:", err)
		http.Error(w, "Could not read lessons from DB", http.StatusInternalServerError)
		return
	}
	type lesson struct {
		data.Entry
		Key string `json:"key"`
	}
	keyedLessons := make([]lesson, 0)
	for k, l := range lessons {
		keyedLessons = append(keyedLessons, lesson{l, k})
	}
	type response struct {
		Cursor  string   `json:"cursor"`
		Lessons []lesson `json:"lessons"`
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response{cursor, keyedLessons}); err != nil {
		log.Println("Could not write json output:", err)
		http.Error(w, "Could not write json", http.StatusInternalServerError)
		return
	}
}

func (s *server) ServeSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		http.Error(w, "empty query", http.StatusBadRequest)
		return
	}
	jsr, err := s.searcher.Search(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type searchResponse struct {
		Query    string
		TookMs   int
		TimedOut bool
		Sources  []source
	}
	sr := searchResponse{
		Query:    q,
		TookMs:   jsr.TookMs,
		TimedOut: jsr.TimedOut,
		Sources:  make([]source, 0),
	}
	for _, hit := range jsr.Hits.Hits {
		sr.Sources = append(sr.Sources, hit.Source)
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

	elasticHostPort := "http://" + os.Getenv("ELASTICSEARCH_SERVICE_HOST") + ":" + os.Getenv("ELASTICSEARCH_SERVICE_PORT")
	s := &server{
		ctx: ctx,
		db:  &datastoreWrapper{client: client},
		searcher: &elasticSearcher{
			httpClient: &http.Client{
				Timeout: time.Second * 2,
			},
			elasticAddress: elasticHostPort,
		},
	}
	http.HandleFunc("/", s.ServeIndex)
	http.HandleFunc("/search", s.ServeSearch)
	http.HandleFunc("/api/lessons", s.APIServeLessons)

	log.Fatal(http.ListenAndServe(*hostPort, nil))
}
