package indexer

import (
	"io"
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
		if _, err := io.WriteString(w, `{"took":11,"errors":false,"items":[{"index":{"_index":"course","_type":"transcript","_id":"AV2O2EyhLu53oBP8SQm_","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"created":true,"status":201}},{"index":{"_index":"course","_type":"transcript","_id":"AV2O2EyhLu53oBP8SQnA","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"created":true,"status":201}}]}`); err != nil {
			t.Fatalf("Could not send test response %v", err)
		}
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

func TestIndexFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, `{"took":11,"errors":true,"items":[{"index":{"_index":"course","_type":"transcript","_id":"AV2O2EyhLu53oBP8SQm_","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"created":true,"status":201}},{"index":{"_index":"course","_type":"transcript","_id":"AV2O2EyhLu53oBP8SQnA","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"created":true,"status":201}}]}`); err != nil {
			t.Fatalf("Could not send test response %v", err)
		}
	}))
	defer ts.Close()

	i := NewElasticIndexer(ts.URL)
	err := i.Index(data.Course{Title: "a title"}, []string{"sentence 1"})
	if err == nil {
		t.Error("Wanted indexing error but got nil")
	}
}
