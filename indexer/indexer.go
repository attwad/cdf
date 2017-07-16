package indexer

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/attwad/cdf/data"
)

type Indexer interface {
	Index(data.Course, []string) error
}

type elasticIndexer struct {
	client *http.Client
	host   string
}

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
	for _, sentence := range sentences {
		jt := transcript{Course: c, Transcript: sentence}
		b, err := json.Marshal(jt)
		if err != nil {
			return err
		}
		js = append(js, seb, string(b))
	}
	// log.Println(strings.Join(js, "\n"))
	r := strings.NewReader(strings.Join(js, "\n"))
	_, err = i.client.Post(i.host+"/_bulk", "application/json", r)
	return err
}
