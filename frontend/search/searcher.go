package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// Source is a source indexed by the search engine.
type Source struct {
	Title      string `json:"title"`
	Lecturer   string `json:"lecturer"`
	Chaire     string `json:"chaire"`
	Type       string `json:"type"`
	TypeTitle  string `json:"type_title"`
	Language   string `json:"lang"`
	URL        string `json:"source_url"`
	Transcript string `json:"transcript"`
}
type hit struct {
	Source Source `json:"_source"`
}
type hits struct {
	Total int   `json:"total"`
	Hits  []hit `json:"hits"`
}

// Response is what gets returned by a Search request.
type Response struct {
	TookMs   int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     hits
}

// Searcher allows free-text search over the transcripts.
type Searcher interface {
	Search(string) (*Response, error)
}

type elasticSearcher struct {
	httpClient     *http.Client
	elasticAddress string
}

// NewElasticSearcher creates a new Searcher connected to the given elastic search instance at the given address.
func NewElasticSearcher(elasticAddress string) Searcher {
	return &elasticSearcher{
		httpClient: &http.Client{
			Timeout: time.Second * 2,
		},
		elasticAddress: elasticAddress,
	}
}

func (e *elasticSearcher) Search(q string) (*Response, error) {
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
	var jsr Response
	if err := json.NewDecoder(resp.Body).Decode(&jsr); err != nil {
		return nil, err
	}
	return &jsr, nil
}
