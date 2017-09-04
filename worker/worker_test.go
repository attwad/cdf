package worker

import (
	"context"
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

type fakeHealthChecker struct {
	healthy bool
	http.Handler
}

func (fhc *fakeHealthChecker) IsHealthy() bool {
	return fhc.healthy
}

type fakePicker struct {
	scheduledCourses map[string]data.Course
	convertedKey     string
	scheduledLength  int
	fullText         string
}

func (p *fakePicker) ScheduleRandom(context.Context, time.Duration) (time.Duration, error) {
	return time.Duration(p.scheduledLength) * time.Second, nil
}

func (p *fakePicker) GetScheduled(context.Context) (map[string]data.Course, error) {
	return p.scheduledCourses, nil
}

func (p *fakePicker) MarkConverted(_ context.Context, key, fullText string) error {
	p.convertedKey = key
	p.fullText = fullText
	return nil
}

type fakeTranscriber struct {
	transcription []transcribe.Transcription
}

func (t *fakeTranscriber) Transcribe(ctx context.Context, lang, path string, hints []string) ([]transcribe.Transcription, error) {
	return t.transcription, nil
}

func (t *fakeTranscriber) ConvertToFLAC(ctx context.Context, soxPath, input string) ([]string, error) {
	return []string{input}, nil
}

type fakeBroker struct {
	balance         int
	getBalanceError error
}

func (b *fakeBroker) GetBalance(ctx context.Context) (int, error) {
	return b.balance, b.getBalanceError
}

func (b *fakeBroker) ChangeBalance(ctx context.Context, delta int) error {
	b.balance -= delta
	return nil
}

type fakeUploader struct {
	uploadedFiles []string
	deletedFiles  []string
}

func (f *fakeUploader) UploadFile(ctx context.Context, r io.Reader, name string) error {
	f.uploadedFiles = append(f.uploadedFiles, name)
	return nil
}

func (f *fakeUploader) Path(base string) string {
	return base
}

func (f *fakeUploader) Delete(ctx context.Context, name string) error {
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
				broker: &fakeBroker{balance: 500},
				picker: &fakePicker{scheduledLength: 10},
			},
			taskScheduled: true,
			wantError:     false,
		}, {
			msg: "not enough balance",
			w: Worker{
				broker: &fakeBroker{balance: 10},
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
					balance:         500,
					getBalanceError: fmt.Errorf("not connected"),
				},
				picker: &fakePicker{},
			},
			taskScheduled: false,
			wantError:     true,
		},
	}
	ctx := context.Background()
	for _, test := range tests {
		taskScheduled, err := test.w.MaybeSchedule(ctx)
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
		health: &fakeHealthChecker{healthy: true},
	}
	ctx := context.Background()
	if err := w.Run(ctx); err != nil {
		t.Errorf("Run: %v", err)
	}
	// Check that we marked the file as completed.
	if got, want := fp.convertedKey, "k1"; got != want {
		t.Errorf("Converted key, got=%q, want=%q", got, want)
	}
	// Check that we saved the transcript.
	if got, want := fp.fullText, "line 1 line 2"; got != want {
		t.Errorf("Saved transcript, got=%q, want=%q", got, want)
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
