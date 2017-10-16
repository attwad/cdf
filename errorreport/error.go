package errorreport

import (
	"context"

	"cloud.google.com/go/errorreporting"
)

// Reporter sends error reports to a log service.
type Reporter interface {
	Report(err error)
	Close() error
}

type stackdriverReporter struct {
	client *errorreporting.Client
}

func (s *stackdriverReporter) Report(err error) {
	s.client.Report(errorreporting.Entry{
		Error: err,
	})
}

func (s *stackdriverReporter) Close() error {
	return s.client.Close()
}

// NewStackdriverReporter creates an error reporter that logs errors to stackdriver.
func NewStackdriverReporter(ctx context.Context, projectID, serviceName string) (Reporter, error) {
	ec, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, err
	}
	return &stackdriverReporter{
		client: ec,
	}, nil
}
