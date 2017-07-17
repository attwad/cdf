package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/attwad/cdf/indexer"
	"github.com/attwad/cdf/money"
	"github.com/attwad/cdf/pick"
	"github.com/attwad/cdf/transcribe"
	"github.com/attwad/cdf/upload"
)

type analyzer struct {
	uploader     upload.FileUploader
	transcriber  transcribe.Transcriber
	broker       money.Broker
	picker       pick.Picker
	indexer      indexer.Indexer
	pricePerTask int
	httpClient   *http.Client
}

func (a *analyzer) Run() error {
	// If we have any money, schedule some tasks.
	balance, err := a.broker.GetBalance()
	if err != nil {
		return err
	}
	log.Println("Balance=", balance, "pricePerTask=", a.pricePerTask)
	if balance < a.pricePerTask {
		return nil
	}
	for balance-a.pricePerTask > 0 {
		log.Println("Enough money to schedule a new task:", balance)
		if err := a.picker.ScheduleRandom(); err != nil {
			return err
		}
		balance -= a.pricePerTask
		if err := a.broker.ChangeBalance(-a.pricePerTask); err != nil {
			return err
		}
		log.Println("New task scheduled")
	}
	// Handle the scheduled tasks.
	courses, err := a.picker.GetScheduled()
	if err != nil {
		return err
	}
	for _, course := range courses {
		// Download file from the web.
		f, tmpCleanup, err := a.downloadToTmpFile(course.AudioLink)
		if err != nil {
			log.Fatal(err)
		}
		defer tmpCleanup()
		// Convert to FLAC.
		flacName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())) + ".flac"
		log.Println("Converting", f.Name(), "to flac @", flacName)
		if err := transcribe.ConvertToFLAC(context.Background(), *soxPath, f.Name(), flacName); err != nil {
			return err
		}
		// Save FLAC to cloud storage.
		if err := a.uploader.UploadFile(f, filepath.Base(f.Name())); err != nil {
			return err
		}
		// Send it to speech recognition.
		t, err := a.transcriber.Transcribe(a.uploader.Path(f.Name()), course.Hints())
		if err != nil {
			return err
		}
		// Save the text output to cloud storage.
		text := make([]string, 0)
		for _, b := range t {
			text = append(text, b.Text)
		}
		textName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())) + ".txt"
		log.Println("Saving text to", textName)
		if err := a.uploader.UploadFile(strings.NewReader(strings.Join(text, " ")), filepath.Base(textName)); err != nil {
			return err
		}
		// Remove FLAC file from cloud storage.
		if err := a.uploader.Delete(f.Name()); err != nil {
			return err
		}
		// Index sentences.
		if err := a.indexer.Index(course, text); err != nil {
			return err
		}
		// Mark the file as converted.
		if err := a.picker.MarkConverted(course); err != nil {
			return err
		}
	}
	// TODO: If tasks get scheduled while analysis was running, it will sleep.
	return nil
}

// downloaddToFile downloads the url target into a temporary file that should be cleaned up by calling the cleanup function returned by this method.
func (a *analyzer) downloadToTmpFile(url string) (*os.File, func(), error) {
	tmpFile, err := ioutil.TempFile("", "cdf-dl")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { os.Remove(tmpFile.Name()) }
	resp, err := a.httpClient.Get(url)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return tmpFile, cleanup, nil
}

var (
	projectID    = flag.String("project_id", "", "Project ID")
	bucket       = flag.String("bucket", "", "Cloud storage bucket")
	soxPath      = flag.String("sox_path", "sox", "SOX binary path")
	elasticHost  = flag.String("elastic_host", "http://localhost:9200", "address of the elasticsearch instance")
	pricePerTask = flag.Int("price_per_task", 100, "Price in cents per task")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	p, err := pick.NewDatastorePicker(*projectID)
	if err != nil {
		log.Fatal(err)
	}
	// p.ScheduleRandom()
	//return
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
	a := analyzer{
		uploader:     u,
		transcriber:  t,
		picker:       p,
		broker:       b,
		indexer:      indexer.NewElasticIndexer(*elasticHost),
		pricePerTask: *pricePerTask,
		// Any download of file shouldn't take more than a few minutes really...
		httpClient: &http.Client{
			Timeout: time.Minute * 5,
		},
	}
	log.Println("Analyzer created, entering loop...")
	for {
		if err := a.Run(); err != nil {
			log.Fatalf("running: %v", err)
		}
		log.Println("Sleeping...")
		time.Sleep(1 * time.Minute)
	}
}
