package main

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/attwad/cdf/data"
)

type fakeDB struct {
}

func (d *fakeDB) GetLessons(ctx context.Context, cursor string) (map[string]data.Entry, string, error) {
	return map[string]data.Entry{"key1": data.Entry{}}, "next cursor", nil
}

func TestIndex(t *testing.T) {
	db := &fakeDB{}
	s := &server{ctx: context.Background(), db: db}
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.ServeIndex(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if !strings.Contains(string(body), "mdl-card") {
		t.Errorf("Expected body to contain an mdl-card but did not: %s", string(body))
	}
	if !strings.Contains(string(body), "next%20cursor") {
		t.Errorf("Expected next cursor link but was not found: %s", string(body))
	}
}

func TestSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `
{
  "took" : 112,
  "timed_out" : false,
  "_shards" : {
    "total" : 6,
    "successful" : 6,
    "failed" : 0
  },
  "hits" : {
    "total" : 2,
    "max_score" : 0.9260096,
    "hits" : [
      {
        "_index" : "course",
        "_type" : "transcript",
        "_id" : "AV2O1vGKLu53oBP8SQm-",
        "_score" : 0.9260096,
        "_source" : {
          "title" : "What was at Stake in the India-China Opium Trade?",
          "lecturer" : "Xavier Paulès",
          "function" : "EHESS, Paris",
          "date" : "23 juin 2017",
          "type" : "Colloque",
          "type_title" : "Inde-Chine : Universalités croisées",
          "chaire" : "Histoire intellectuelle de la Chine",
          "lang" : "",
          "transcript" : "chez les humains à tout particulièrement bien se nous ramène examen à cette distinction entre l'instant et l'intelligence distinction qui a été proposé par virement personne mais il nous ramène les Street geometry L'Express d'une certaine façon donc ça c'est les trois conclusion que je voulais te dire et justement pour qu'on puisse continuer à réfléchir tous ensemble. Merci Antoine d'avoir donné la place à Darwin Collège de France puisque d'Antoine à faire entrer Darwin au Collège de France une fois plus et je voudrais de nouveaux vous remercie de tous pour la façon dont vous avez participé et avec pascal mans essayer de rattraper votre retard scolaire en tout cas le mien j'en ai attrapé un beau mec même si c'est quelque chose de tellement irrattrapable je remercie je vous souhaite une bonne soirée"
        }
      },
      {
        "_index" : "course",
        "_type" : "transcript",
        "_id" : "AV1BNNE-1rOPjanaTvb9",
        "_score" : 0.9256437,
        "_source" : {
          "title" : "What was at Stake in the India-China Opium Trade?",
          "lecturer" : "Xavier Paulès",
          "function" : "EHESS, Paris",
          "date" : "23 juin 2017",
          "type" : "Colloque",
          "type_title" : "Inde-Chine : Universalités croisées",
          "chaire" : "Histoire intellectuelle de la Chine",
          "lang" : "",
          "transcript" : "chez les humains à tout particulièrement bien se nous ramène examen à cette distinction entre l'instant et l'intelligence distinction qui a été proposé par virement personne mais il nous ramène les Street geometry L'Express d'une certaine façon donc ça c'est les trois conclusion que je voulais te dire et justement pour qu'on puisse continuer à réfléchir tous ensemble. Merci Antoine d'avoir donné la place à Darwin Collège de France puisque d'Antoine à faire entrer Darwin au Collège de France une fois plus et je voudrais de nouveaux vous remercie de tous pour la façon dont vous avez participé et avec pascal mans essayer de rattraper votre retard scolaire en tout cas le mien j'en ai attrapé un beau mec même si c'est quelque chose de tellement irrattrapable je remercie je vous souhaite une bonne soirée"
        }
      }
    ]
  }
}
`)
	}))
	defer ts.Close()

	db := &fakeDB{}
	s := &server{
		ctx: context.Background(),
		searcher: &elasticSearcher{
			httpClient: &http.Client{
				Timeout: time.Second * 2,
			},
			elasticAddress: ts.URL,
		},
		db: db,
	}
	req := httptest.NewRequest("GET", "/search?q=myquery", nil)
	w := httptest.NewRecorder()
	s.ServeSearch(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if !strings.Contains(string(body), "took 112ms") {
		t.Errorf("Expected body to contain 'took 112ms' but did not: %s", string(body))
	}
}
