package worker

import (
	"context"
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

// Worker does the actual job of checking the balance, scheduling tasks, downloading audio files, transcribing them, etc.
type Worker struct {
	uploader     upload.FileUploader
	transcriber  transcribe.Transcriber
	broker       money.Broker
	picker       pick.Picker
	indexer      indexer.Indexer
	pricePerTask int
	soxPath      string
	httpClient   *http.Client
}

// NewGCPWorker creates a new worker that does its work using Google Cloud Platform.
func NewGCPWorker(u upload.FileUploader, t transcribe.Transcriber, m money.Broker, p pick.Picker, i indexer.Indexer, pricePerTask int, soxPath string) *Worker {
	return &Worker{
		u, t, m, p, i, pricePerTask, soxPath,
		// Any download of file shouldn't take more than a few minutes really...
		&http.Client{
			Timeout: time.Minute * 5,
		},
	}
}

// Run checks for scheduled tasks and handle all of them if any.
func (w *Worker) Run() error {
	// Handle the scheduled tasks.
	courses, err := w.picker.GetScheduled()
	if err != nil {
		return err
	}
	for key, course := range courses {
		// Download file from the web.
		f, tmpCleanup, err := w.downloadToTmpFile(course.AudioLink)
		if err != nil {
			log.Fatal(err)
		}
		defer tmpCleanup()
		// Convert to FLAC.
		flacName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())) + ".flac"
		log.Println("Converting", f.Name(), "to flac @", flacName)
		if err := transcribe.ConvertToFLAC(context.Background(), w.soxPath, f.Name(), flacName); err != nil {
			return err
		}
		// Save FLAC to cloud storage.
		if err := w.uploader.UploadFile(f, filepath.Base(f.Name())); err != nil {
			return err
		}
		// Send it to speech recognition.
		t, err := w.transcriber.Transcribe(course.Language, w.uploader.Path(f.Name()), course.Hints())
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
		if err := w.uploader.UploadFile(strings.NewReader(strings.Join(text, " ")), filepath.Base(textName)); err != nil {
			return err
		}
		// Remove FLAC file from cloud storage.
		if err := w.uploader.Delete(f.Name()); err != nil {
			return err
		}
		// Index sentences.
		if err := w.indexer.Index(course, text); err != nil {
			return err
		}
		// Mark the file as converted.
		if err := w.picker.MarkConverted(key); err != nil {
			return err
		}
	}
	return nil
}

// downloaddToFile downloads the url target into a temporary file that should be cleaned up by calling the cleanup function returned by this method.
func (w *Worker) downloadToTmpFile(url string) (*os.File, func(), error) {
	tmpFile, err := ioutil.TempFile("", "cdf-dl")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { os.Remove(tmpFile.Name()) }
	resp, err := w.httpClient.Get(url)
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

// MaybeSchedule checks the current balance and schedule new audio tracks to be
// transcribed if the balance is > w.pricePerTask.
// Returns whether new tasks were scheduled.
func (w *Worker) MaybeSchedule() (bool, error) {
	// If we have any money, schedule some tasks.
	balance, err := w.broker.GetBalance()
	if err != nil {
		return false, err
	}
	log.Println("Balance=", balance, "pricePerTask=", w.pricePerTask)
	if balance < w.pricePerTask {
		return false, nil
	}
	for balance-w.pricePerTask > 0 {
		log.Println("Enough money to schedule a new task:", balance)
		if err := w.picker.ScheduleRandom(); err != nil {
			return false, err
		}
		log.Println("New task scheduled")
		balance -= w.pricePerTask
		if err := w.broker.ChangeBalance(-w.pricePerTask); err != nil {
			return false, err
		}
		log.Println("Decreased balance")
	}
	return true, nil
}
