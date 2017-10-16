package logging

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/logging"
)

// NewLogger returns a new local or remote logger.
func NewLogger(ctx context.Context, projectID, logType string) (infoLog, alertLog *log.Logger, cleanup func(), err error) {
	if logType != "local" && logType != "stackdriver" {
		return nil, nil, nil, fmt.Errorf("logType must be == 'local' or 'stackdriver', was=%q", logType)
	}
	if logType == "local" {
		return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile), log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile), func() {}, nil
	}

	loggingClient, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create client: %v", err)
	}
	loggingClient.OnError = func(e error) {
		log.Println("[ERROR]: logging:", e)
	}
	def := func() {
		loggingClient.Close()
	}
	logger := loggingClient.Logger("worker")
	infoLog = logger.StandardLogger(logging.Info)
	infoLog.SetFlags(log.Lshortfile)
	alertLog = logger.StandardLogger(logging.Alert)
	alertLog.SetFlags(log.Lshortfile)

	return infoLog, alertLog, def, nil
}
