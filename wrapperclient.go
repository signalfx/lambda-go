package sfxlambda

import (
	"context"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"strings"
	"time"
)

var handlerFuncWrapperClient *sfxclient.HTTPSink

func init() {
	handlerFuncWrapperClient = sfxclient.NewHTTPSink()
	if handlerFuncWrapperClient.AuthToken = os.Getenv("SIGNALFX_AUTH_TOKEN"); handlerFuncWrapperClient.AuthToken == "" {
		log.Fatalf("Required environment variable SIGNALFX_AUTH_TOKEN is not set")
	}
	if os.Getenv("SIGNALFX_INGEST_ENDPOINT") != "" {
		if ingestURL, err := url.Parse(os.Getenv("SIGNALFX_INGEST_ENDPOINT")); err == nil {
			if ingestURL, err = ingestURL.Parse("v2/datapoint"); err == nil {
				handlerFuncWrapperClient.DatapointEndpoint = ingestURL.String()
			} else {
				log.Fatalf("Error parsing environment variable SIGNALFX_INGEST_ENDPOINT url value %s: %+v", os.Getenv("SIGNALFX_INGEST_ENDPOINT"), err)
			}
		} else {
			log.Fatalf("Error parsing ingest url path v2/datapoint: %+v", err)
		}
	}
	if os.Getenv("SIGNALFX_SEND_TIMEOUT") != "" {
		if timeout, err := time.ParseDuration(strings.TrimSpace(os.Getenv("SIGNALFX_SEND_TIMEOUT")) + "s"); err == nil {
			handlerFuncWrapperClient.Client.Timeout = timeout
		} else {
			log.Fatalf("Error parsing environment variable SIGNALFX_SEND_TIMEOUT timeout value %s: %+v", os.Getenv("SIGNALFX_SEND_TIMEOUT"), err)
		}
	}
}

var sendDatapoint = func(ctx context.Context, dp *datapoint.Datapoint) {
	if ctx == nil {
		ctx = context.Background()
	}
	if &dp.Timestamp == nil {
		dp.Timestamp = time.Now()
	}
	go func(ctx context.Context, dp *datapoint.Datapoint) {
		if err := handlerFuncWrapperClient.AddDatapoints(ctx, []*datapoint.Datapoint{dp}); err != nil {
			log.Errorf("Error sending datapoint to SignalFx: %+v", err)
		}
	}(ctx, dp)
}
