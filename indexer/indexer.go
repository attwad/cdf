package indexer

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/attwad/cdf/data"
)

// Indexer handles indexing of a course's transcript.
type Indexer interface {
	Index(data.Course, []string) error
}

type elasticIndexer struct {
	client *http.Client
	host   string
}

// NewElasticIndexer creates a new Indexer connected to elastic search.
func NewElasticIndexer(host string) Indexer {
	return &elasticIndexer{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		host: host,
	}
}

type entry struct {
	Index indexEntry `json:"index"`
}

type indexEntry struct {
	Index string `json:"_index"`
	Type  string `json:"_type"`
}

type transcript struct {
	data.Course
	Serial     int
	Transcript string `json:"transcript"`
}

func (i *elasticIndexer) Index(c data.Course, sentences []string) error {
	js := make([]string, 0)
	e := entry{Index: indexEntry{Index: "course", Type: "transcript"}}
	eb, err := json.Marshal(e)
	if err != nil {
		return err
	}
	seb := string(eb)
	for i, sentence := range sentences {
		jt := transcript{Course: c, Transcript: sentence, Serial: i}
		b, err2 := json.Marshal(jt)
		if err2 != nil {
			return err
		}
		js = append(js, seb, string(b))
	}
	r := strings.NewReader(strings.Join(js, "\n") + "\n")
	resp, err := i.client.Post(i.host+"/_bulk", "application/json", r)
	if err != nil {
		return err
	}
	// TODO: Parse response.
	defer resp.Body.Close()
	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Println("Index response:", string(rb))
	return nil
}
