package transcribe

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	longrunningpb "google.golang.org/genproto/googleapis/longrunning"
)

// ConvertToFLAC converts the input audio file into a FLAC audio file as the output filename using the program sox.
func ConvertToFLAC(ctx context.Context, soxPath, input, output string) error {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, soxPath, input, output, "channels", "1", "rate", "16k").Run()
}

// Transcription contains what was said with a given confidence score for the overall transcription.
type Transcription struct {
	Text       string
	confidence float32
}

// Transcriber allows transcription of an audio file.
type Transcriber interface {
	Transcribe(gcsPath string, hints []string) ([]Transcription, error)
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

func (g *gSpeechTranscriber) Transcribe(gcsURI string, hints []string) ([]Transcription, error) {
	opName, err := g.sendGCS(gcsURI, hints)
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

func (g *gSpeechTranscriber) sendGCS(gcsURI string, hints []string) (string, error) {
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_FLAC,
			SampleRateHertz: 16000,
			LanguageCode:    "fr-FR",
			SpeechContexts: []*speechpb.SpeechContext{
				{Phrases: hints},
			},
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}

	op, err := g.client.LongRunningRecognize(g.ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}
