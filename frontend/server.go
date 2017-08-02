package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

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
	ctx context.Context
	db  dbWrapper
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
		log.Println("Could not read lessosn from db:", err)
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

func main() {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		ctx: ctx,
		db:  &datastoreWrapper{client: client},
	}
	http.HandleFunc("/", s.ServeIndex)

	log.Fatal(http.ListenAndServe(*hostPort, nil))
}
