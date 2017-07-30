package worker

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/attwad/cdf/data"
	"github.com/attwad/cdf/transcribe"
)

type fakePicker struct {
	scheduledCourses map[string]data.Course
	convertedKey     string
	scheduledLength  int
}

func (p *fakePicker) ScheduleRandom(int) (int, error) {
	return p.scheduledLength, nil
}

func (p *fakePicker) GetScheduled() (map[string]data.Course, error) {
	return p.scheduledCourses, nil
}

func (p *fakePicker) MarkConverted(key string) error {
	p.convertedKey = key
	return nil
}

type fakeTranscriber struct {
	transcription []transcribe.Transcription
}

func (t *fakeTranscriber) Transcribe(lang, path string, hints []string) ([]transcribe.Transcription, error) {
	return t.transcription, nil
}

func (t *fakeTranscriber) ConvertToFLAC(soxPath, input string) ([]string, error) {
	return []string{input}, nil
}

type fakeBroker struct {
	balance         int
	getBalanceError error
}

func (b *fakeBroker) GetBalance() (int, error) {
	return b.balance, b.getBalanceError
}

func (b *fakeBroker) ChangeBalance(delta int) error {
	b.balance -= delta
	return nil
}

type fakeUploader struct {
	uploadedFiles []string
	deletedFiles  []string
}

func (f *fakeUploader) UploadFile(r io.Reader, name string) error {
	f.uploadedFiles = append(f.uploadedFiles, name)
	return nil
}

func (f *fakeUploader) Path(base string) string {
	return base
}

func (f *fakeUploader) Delete(name string) error {
	f.deletedFiles = append(f.deletedFiles, name)
	return nil
}

type fakeIndexer struct {
	indexedText string
}

func (f *fakeIndexer) Index(course data.Course, text []string) error {
	f.indexedText += strings.Join(text, "")
	return nil
}

func TestMaybeSchedule(t *testing.T) {
	var tests = []struct {
		msg           string
		w             Worker
		taskScheduled bool
		wantError     bool
	}{
		{
			msg: "balance ok",
			w: Worker{
				broker: &fakeBroker{balance: 50},
				picker: &fakePicker{scheduledLength: 10},
			},
			taskScheduled: true,
			wantError:     false,
		}, {
			msg: "not enough balance",
			w: Worker{
				broker: &fakeBroker{balance: 0},
				picker: &fakePicker{},
			},
			taskScheduled: false,
			wantError:     false,
		}, {
			msg: "nothing to schedule",
			w: Worker{
				broker: &fakeBroker{balance: 10},
				picker: &fakePicker{},
			},
			taskScheduled: false,
			wantError:     false,
		}, {
			msg: "error checking balance",
			w: Worker{
				broker: &fakeBroker{
					balance:         50,
					getBalanceError: fmt.Errorf("not connected"),
				},
				picker: &fakePicker{},
			},
			taskScheduled: false,
			wantError:     true,
		},
	}
	for _, test := range tests {
		taskScheduled, err := test.w.MaybeSchedule()
		if got, want := taskScheduled, test.taskScheduled; got != want {
			t.Errorf("[%s] task scheduled, got=%t, want=%t", test.msg, got, want)
		}
		if got, want := err != nil, test.wantError; got != want {
			t.Errorf("[%s]wantError, got=%t, want=%t", test.msg, got, want)
		}
	}
}

func TestDownloadToTmpFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	w := Worker{
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
	}
	_, cleanup, err := w.downloadToTmpFile(ts.URL)
	if err != nil {
		t.Fatalf("downloadToTmpFile: %v", err)
	}
	defer cleanup()
}

func TestRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()
	fp := &fakePicker{
		scheduledCourses: map[string]data.Course{"k1": {AudioLink: ts.URL}},
	}
	fu := &fakeUploader{uploadedFiles: make([]string, 0)}
	fi := &fakeIndexer{}
	transcript := []transcribe.Transcription{
		{Text: "line 1"},
		{Text: "line 2"},
	}
	w := Worker{
		picker:      fp,
		transcriber: &fakeTranscriber{transcription: transcript},
		uploader:    fu,
		indexer:     fi,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
	}
	if err := w.Run(); err != nil {
		t.Errorf("Run: %v", err)
	}
	// Check that we marked the file as completed.
	if got, want := fp.convertedKey, "k1"; got != want {
		t.Errorf("Converted key, got=%q, want=%q", got, want)
	}
	// Check that we saved the flac and text file with the transcript.
	if got, want := len(fu.uploadedFiles), 2; got != want {
		t.Errorf("Num saved files, got=%d, want=%d", got, want)
	}
	// Check that we deleted the flac file.
	if got, want := len(fu.deletedFiles), 1; got != want {
		t.Errorf("Num deleted files, got=%d, want=%d", got, want)
	}
	// Check that we indexed the transcript.
	if got, want := fi.indexedText, "line 1line 2"; got != want {
		t.Errorf("Num indexed text, got=%q, want=%q", got, want)
	}
}
