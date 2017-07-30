package main

import (
	"log"

	"github.com/attwad/cdf/indexer"
	"github.com/attwad/cdf/testdata"
)

func main() {
	if err := indexer.NewElasticIndexer("http://localhost:9200").Index(
		testdata.CreateCourse(), testdata.CreateTranscript()); err != nil {
		log.Fatal(err)
	}
}
