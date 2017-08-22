package main

import (
	"flag"
	"log"
	"time"

	"github.com/attwad/cdf/indexer"
	"github.com/attwad/cdf/money"
	"github.com/attwad/cdf/pick"
	"github.com/attwad/cdf/transcribe"
	"github.com/attwad/cdf/upload"
	"github.com/attwad/cdf/worker"
)

var (
	projectID      = flag.String("project_id", "", "Project ID")
	bucket         = flag.String("bucket", "", "Cloud storage bucket")
	soxPath        = flag.String("sox_path", "sox", "SOX binary path")
	elasticAddress = flag.String("elastic_address", "http://elastic:9200", "HTTP address to elastic instance")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	p, err := pick.NewDatastorePicker(*projectID)
	if err != nil {
		log.Fatal(err)
	}
	u, err := upload.NewGCSFileUploader(*bucket)
	if err != nil {
		log.Fatal(err)
	}
	t, err := transcribe.NewGSpeechTranscriber()
	if err != nil {
		log.Fatal(err)
	}
	b, err := money.NewDatastoreBroker(*projectID)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Will connect to elastic instance @", *elasticAddress)
	a := worker.NewGCPWorker(
		u,
		t,
		b,
		p,
		indexer.NewElasticIndexer(*elasticAddress),
		*soxPath)
	log.Println("Analyzer created, entering loop...")
	for {
		if err := a.Run(); err != nil {
			log.Fatalf("running: %v", err)
		}
		hasNew, err := a.MaybeSchedule()
		if err != nil {
			log.Fatalf("Scheduling new tasks: %v", err)
		}
		// Only sleep if we have nothing scheduled.
		if !hasNew {
			log.Println("Sleeping...")
			time.Sleep(1 * time.Minute)
		}
	}
}
