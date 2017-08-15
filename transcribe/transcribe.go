package transcribe

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"
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
	Transcribe(lang, path string, hints []string) ([]Transcription, error)
	ConvertToFLAC(soxPath, input string) ([]string, error)
}

type gSpeechTranscriber struct {
	client *speech.Client
	ctx    context.Context
}

// NewGSpeechTranscriber creates a new transcriber using the Google Speech API.
func NewGSpeechTranscriber() (Transcriber, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &gSpeechTranscriber{
		client: client,
		ctx:    ctx,
	}, nil
}

func (g *gSpeechTranscriber) Transcribe(lang, gcsURI string, hints []string) ([]Transcription, error) {
	opName, err := g.sendGCS(lang, gcsURI, hints)
	if err != nil {
		return nil, err
	}

	resp, err := g.wait(opName)
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

func (g *gSpeechTranscriber) wait(opName string) (*speechpb.LongRunningRecognizeResponse, error) {
	opClient := longrunningpb.NewOperationsClient(g.client.Connection())
	var op *longrunningpb.Operation
	var err error
	for {
		op, err = opClient.GetOperation(g.ctx, &longrunningpb.GetOperationRequest{
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

func (g *gSpeechTranscriber) sendGCS(lang, gcsURI string, hints []string) (string, error) {
	// Not requesting per work offset via "enableWordTimeOffsets": true in the config
	// as I am not sure how useful it would be...
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_FLAC,
			SampleRateHertz: 16000,
			LanguageCode:    language.Make(lang).String(), // Must be a BCP-47 identifier.
			SpeechContexts: []*speechpb.SpeechContext{
				{Phrases: hints},
			},
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}
	log.Println("Sending gspeech request", req)

	op, err := g.client.LongRunningRecognize(g.ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}

// ConvertToFLAC converts the input audio file into a FLAC audio file as the output filename using the program sox.
// Returns the output paths.
func (g *gSpeechTranscriber) ConvertToFLAC(soxPath, input string) ([]string, error) {
	ctx, cancel := context.WithTimeout(g.ctx, 120*time.Second)
	defer cancel()
	flacName := input + ".flac"
	log.Println("Converting", input, "to flac @", flacName)
	// Convert input to mono FLAC, split output in chunks of 59min as GCP Speech
	// API supports max 3h chunks.
	// 10790 = 2.99 hours.
	err := exec.CommandContext(ctx, soxPath, "-t", "mp3", input, flacName, "channels", "1", "rate", "16k", "trim", "0", "10790", ":", "newfile", ":", "restart").Run()
	if err != nil {
		return nil, err
	}
	return filepath.Glob(input + "*.flac")
}
