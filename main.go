package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/attwad/cdf/errorreport"
	"github.com/attwad/cdf/health"
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
	ctx := context.Background()

	er, err := errorreport.NewStackdriverReporter(ctx, *projectID, "worker")
	if err != nil {
		log.Fatalf("Creating error reporting client: %v", err)
	}
	defer er.Close()

	p, err := pick.NewDatastorePicker(ctx, *projectID)
	if err != nil {
		log.Fatal(err)
	}
	u, err := upload.NewGCSFileUploader(ctx, *bucket)
	if err != nil {
		log.Fatal(err)
	}
	t, err := transcribe.NewGSpeechTranscriber(ctx)
	if err != nil {
		log.Fatal(err)
	}
	b, err := money.NewDatastoreBroker(ctx, *projectID)
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
		*soxPath,
		health.NewElasticHealthChecker(*elasticAddress))
	log.Println("Analyzer created, entering loop...")
	for {
		if err := a.Run(ctx); err != nil {
			log.Println("[ERROR]: running:", err)
			er.Report(fmt.Errorf("Running: %v", err))
		}
		hasNew, err := a.MaybeSchedule(ctx)
		if err != nil {
			log.Fatalf("Scheduling new tasks: %v", err)
			er.Report(fmt.Errorf("Scheduling new task: %v", err))
		}
		// Only sleep if we have nothing scheduled.
		if !hasNew {
			log.Println("Sleeping...")
			time.Sleep(1 * time.Minute)
		}
	}
}
