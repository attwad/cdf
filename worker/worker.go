package worker

import (
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
	uploader    upload.FileUploader
	transcriber transcribe.Transcriber
	broker      money.Broker
	picker      pick.Picker
	indexer     indexer.Indexer
	soxPath     string
	httpClient  *http.Client
}

// NewGCPWorker creates a new worker that does its work using Google Cloud Platform.
func NewGCPWorker(u upload.FileUploader, t transcribe.Transcriber, m money.Broker, p pick.Picker, i indexer.Indexer, soxPath string) *Worker {
	return &Worker{
		u, t, m, p, i, soxPath,
		// Any download of file shouldn't take more than a few minutes really...
		&http.Client{
			Timeout: time.Minute * 30,
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
	// TODO: Add elasticsearch health check.
	for key, course := range courses {
		// Download file from the web.
		log.Println("Downloading", course.AudioLink, "to tmp file")
		f, tmpCleanup, err := w.downloadToTmpFile(course.AudioLink)
		if err != nil {
			return err
		}
		defer tmpCleanup()
		// Convert to FLAC.
		log.Println("Converting to flac")
		paths, err := w.transcriber.ConvertToFLAC(w.soxPath, f.Name())
		if err != nil {
			return err
		}
		log.Println("FLAC files:", paths)
		for _, flac := range paths {
			flacReader, err := os.Open(flac)
			if err != nil {
				return err
			}
			defer flacReader.Close()
			// Save FLAC to cloud storage.
			log.Println("Saving flac to could storage")
			if err := w.uploader.UploadFile(flacReader, filepath.Base(flac)); err != nil {
				return err
			}
			// Send it to speech recognition.
			log.Println("Transcribing audio")
			t, err := w.transcriber.Transcribe(course.Language, w.uploader.Path(filepath.Base(flac)), course.Hints())
			if err != nil {
				return err
			}
			// Save the text output to cloud storage.
			text := make([]string, 0)
			for _, b := range t {
				text = append(text, b.Text)
			}
			textName := filepath.Base(course.AudioLink) + ".txt"
			log.Println("Saving text to", textName)
			log.Println(text)
			if err := w.uploader.UploadFile(strings.NewReader(strings.Join(text, " ")), filepath.Base(textName)); err != nil {
				return err
			}
			// Remove FLAC file from cloud storage.
			// TODO: defer and panic on error?
			log.Println("Deleting flac from cloud storage")
			if err := w.uploader.Delete(filepath.Base(flac)); err != nil {
				return err
			}
			// Index sentences.
			log.Println("Indexing text")
			if err := w.indexer.Index(course, text); err != nil {
				return err
			}
		}
		// Mark the file as converted.
		log.Println("Marking", course.AudioLink, "as converted")
		if err := w.picker.MarkConverted(key); err != nil {
			return err
		}
	}
	return nil
}

// downloadToTmpFile downloads the url target into a temporary file that should be cleaned up by calling the cleanup function returned by this method.
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
// transcribed.
// Simple greedy algorithm, should use dynamic programming if I want to
// optimize for the number of courses converted vs pure length.
// Returns whether new tasks were scheduled.
func (w *Worker) MaybeSchedule() (bool, error) {
	// Get our current balance.
	balance, err := w.broker.GetBalance()
	if err != nil {
		return false, err
	}
	log.Println("Balance=", balance)
	if balance <= 0 {
		return false, nil
	}
	length, err := w.picker.ScheduleRandom(balance)
	if err != nil {
		return false, err
	}
	if length <= 0 {
		return false, nil
	}
	log.Println("New task scheduled")
	if err := w.broker.ChangeBalance(-length); err != nil {
		return false, err
	}
	log.Println("Decreased balance")
	return true, nil
}
