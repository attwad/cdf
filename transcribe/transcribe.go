package transcribe

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"

	"golang.org/x/text/language"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	longrunningpb "google.golang.org/genproto/googleapis/longrunning"
)

// Transcription contains what was said with a given confidence score for the overall transcription.
type Transcription struct {
	Text       string
	confidence float32
}

// Transcriber allows transcription of an audio file.
type Transcriber interface {
	Transcribe(ctx context.Context, lang, path string, hints []string) ([]Transcription, error)
	ConvertToFLAC(ctx context.Context, soxPath, input string) ([]string, error)
}

type gSpeechTranscriber struct {
	client *speech.Client
}

// NewGSpeechTranscriber creates a new transcriber using the Google Speech API.
func NewGSpeechTranscriber(ctx context.Context) (Transcriber, error) {
	client, err := speech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &gSpeechTranscriber{
		client: client,
	}, nil
}

func (g *gSpeechTranscriber) Transcribe(ctx context.Context, lang, gcsURI string, hints []string) ([]Transcription, error) {
	opName, err := g.sendGCS(ctx, lang, gcsURI, hints)
	if err != nil {
		return nil, err
	}

	resp, err := g.wait(ctx, opName)
	if err != nil {
		return nil, err
	}

	transcriptions := make([]Transcription, 0)
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			transcriptions = append(transcriptions, Transcription{
				Text:       alt.Transcript,
				confidence: alt.Confidence,
			})
		}
	}
	return transcriptions, nil
}

func (g *gSpeechTranscriber) wait(ctx context.Context, opName string) (*speechpb.LongRunningRecognizeResponse, error) {
	opClient := longrunningpb.NewOperationsClient(g.client.Connection())
	var op *longrunningpb.Operation
	var err error
	for {
		op, err = opClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{
			Name: opName,
		})
		if err != nil {
			return nil, err
		}
		if op.Done {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	switch {
	case op.GetError() != nil:
		return nil, fmt.Errorf("received error in response: %v", op.GetError())
	case op.GetResponse() != nil:
		var resp speechpb.LongRunningRecognizeResponse
		if err := proto.Unmarshal(op.GetResponse().Value, &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	}

	// should never happen.
	return nil, errors.New("no response")
}

func (g *gSpeechTranscriber) sendGCS(ctx context.Context, lang, gcsURI string, hints []string) (string, error) {
	// Take a language and default to French if ends up undefined.
	var l = language.Make(lang)
	if l == language.Und {
		log.Println("Language", lang, "was undefined, defaulting to French")
		l = language.French
	}
	// Not requesting per work offset via "enableWordTimeOffsets": true in the config
	// as I am not sure how useful it would be...
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_FLAC,
			SampleRateHertz: 16000,
			LanguageCode:    l.String(), // Must be a BCP-47 identifier.
			SpeechContexts: []*speechpb.SpeechContext{
				{Phrases: hints},
			},
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}
	log.Println("Sending gspeech request", req)

	op, err := g.client.LongRunningRecognize(ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}

// ConvertToFLAC converts the input audio file into a FLAC audio file as the output filename using the program sox.
// Returns the output paths.
func (g *gSpeechTranscriber) ConvertToFLAC(ctx context.Context, soxPath, input string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	flacName := input + ".flac"
	log.Println("Converting", input, "to flac @", flacName)
	// Convert input to mono FLAC, split output in chunks of 3H as GCP Speech
	// API supports max 3H chunks.
	// 10790 = 2.99 hours.
	err := exec.CommandContext(ctx, soxPath, "-t", "mp3", input, flacName, "channels", "1", "rate", "16k", "trim", "0", "10790", ":", "newfile", ":", "restart").Run()
	if err != nil {
		return nil, err
	}
	return filepath.Glob(input + "*.flac")
}
