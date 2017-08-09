package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/attwad/cdf/data"
	"github.com/attwad/cdf/frontend/search"

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

type server struct {
	ctx      context.Context
	db       dbWrapper
	searcher search.Searcher
}

type dbWrapper interface {
	GetLessons(ctx context.Context, cursorStr string) ([]data.Entry, string, error)
}

type datastoreWrapper struct {
	client *datastore.Client
}

func (d *datastoreWrapper) GetLessons(ctx context.Context, cursorStr string) ([]data.Entry, string, error) {
	lessons := make([]data.Entry, 0)
	query := datastore.NewQuery("Entry").Order("-Scraped").Limit(pageSize)
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
		_, err := it.Next(&e)
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
		lessons = append(lessons, e)
	}
}

func (s *server) APIServeLessons(w http.ResponseWriter, r *http.Request) {
	lessons, cursor, err := s.db.GetLessons(s.ctx, r.URL.Query().Get("cursor"))
	if err != nil {
		log.Println("Could not read lessons from db:", err)
		http.Error(w, "Could not read lessons from DB", http.StatusInternalServerError)
		return
	}
	type response struct {
		Cursor  string       `json:"cursor"`
		Lessons []data.Entry `json:"lessons"`
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response{cursor, lessons}); err != nil {
		log.Println("Could not write json output:", err)
		http.Error(w, "Could not write json", http.StatusInternalServerError)
		return
	}
}

/*
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
*/
func main() {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}

	elasticHostPort := "http://" + os.Getenv("ELASTICSEARCH_SERVICE_HOST") + ":" + os.Getenv("ELASTICSEARCH_SERVICE_PORT")
	s := &server{
		ctx:      ctx,
		db:       &datastoreWrapper{client: client},
		searcher: search.NewElasticSearcher(elasticHostPort),
	}
	http.HandleFunc("/api/lessons", s.APIServeLessons)

	log.Fatal(http.ListenAndServe(*hostPort, nil))
}
