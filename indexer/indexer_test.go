package indexer

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attwad/cdf/data"
)

func TestIndex(t *testing.T) {
	title := "A lesson"
	sentences := []string{"sentence 1", "sentence 2"}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Could not read request sent to server %v", err)
			return
		}
		defer r.Body.Close()
		s := string(b)
		if !strings.Contains(s, title) {
			t.Errorf("Missing %q in request sent to server", title)
		}
		for _, sentence := range sentences {
			if !strings.Contains(s, sentence) {
				t.Errorf("Missing %q in request sent to server", sentence)
			}
		}
	}))
	defer ts.Close()

	i := NewElasticIndexer(ts.URL)
	if err := i.Index(data.Course{Title: title}, sentences); err != nil {
		t.Errorf("Indexing course: %v", err)
	}
}
