package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/attwad/cdf/logging"

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
	logType        = flag.String("log_type", "local", "Type of logger to use: 'local' or 'strackdriver'")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	infoLog, alertLog, cleanup, err := logging.NewLogger(ctx, *projectID, *logType)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer cleanup()
	alertLog.Panicf("woops")

	p, err := pick.NewDatastorePicker(ctx, *projectID)
	if err != nil {
		alertLog.Panic(err)
	}
	u, err := upload.NewGCSFileUploader(ctx, *bucket)
	if err != nil {
		alertLog.Panic(err)
	}
	t, err := transcribe.NewGSpeechTranscriber(ctx)
	if err != nil {
		alertLog.Panic(err)
	}
	b, err := money.NewDatastoreBroker(ctx, *projectID)
	if err != nil {
		alertLog.Panic(err)
	}
	infoLog.Println("Will connect to elastic instance @", *elasticAddress)
	a := worker.NewGCPWorker(
		u,
		t,
		b,
		p,
		indexer.NewElasticIndexer(*elasticAddress),
		*soxPath,
		health.NewElasticHealthChecker(*elasticAddress),
		infoLog)
	infoLog.Println("Analyzer created, entering loop...")
	for {
		if err := a.Run(ctx); err != nil {
			alertLog.Panicf("running: %v", err)
		}
		hasNew, err := a.MaybeSchedule(ctx)
		if err != nil {
			alertLog.Panicf("Scheduling new tasks: %v", err)
		}
		// Only sleep if we have nothing scheduled.
		if !hasNew {
			infoLog.Println("Sleeping...")
			time.Sleep(1 * time.Minute)
		}
	}
}
