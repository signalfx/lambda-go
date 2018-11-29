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

const (
	sfxAuthToken = "SIGNALFX_AUTH_TOKEN"
	sfxIngestEndpoint = "SIGNALFX_INGEST_ENDPOINT"
	sfxSendTimeoutSeconds= "SIGNALFX_SEND_TIMEOUT_SECONDS"
	)

func init() {
	handlerFuncWrapperClient = sfxclient.NewHTTPSink()
	if handlerFuncWrapperClient.AuthToken = os.Getenv(sfxAuthToken); handlerFuncWrapperClient.AuthToken == "" {
		log.Fatalf("Required environment variable %s is not set", sfxAuthToken)
	}
	if os.Getenv(sfxIngestEndpoint) != "" {
		if ingestURL, err := url.Parse(os.Getenv(sfxIngestEndpoint)); err == nil {
			if ingestURL, err = ingestURL.Parse("v2/datapoint"); err == nil {
				handlerFuncWrapperClient.DatapointEndpoint = ingestURL.String()
			} else {
				log.Fatalf("Error parsing environment variable %s url value %s: %+v", sfxIngestEndpoint, os.Getenv(sfxIngestEndpoint), err)
			}
		} else {
			log.Fatalf("Error parsing ingest url path v2/datapoint: %+v", err)
		}
	}
	if os.Getenv(sfxSendTimeoutSeconds) != "" {
		if timeout, err := time.ParseDuration(strings.TrimSpace(os.Getenv(sfxSendTimeoutSeconds)) + "s"); err == nil {
			handlerFuncWrapperClient.Client.Timeout = timeout
		} else {
			log.Fatalf("Error parsing environment variable %s timeout value %s: %+v", sfxSendTimeoutSeconds, os.Getenv(sfxSendTimeoutSeconds), err)
		}
	}
}

var sendDatapoint = func(ctx context.Context, dp *datapoint.Datapoint) {
	if ctx == nil {
		ctx = context.Background()
	}
	if dp.Timestamp.IsZero() {
		dp.Timestamp = time.Now()
	}
	go func(ctx context.Context, dp *datapoint.Datapoint) {
		if err := handlerFuncWrapperClient.AddDatapoints(ctx, []*datapoint.Datapoint{dp}); err != nil {
			log.Errorf("Error sending datapoint to SignalFx: %+v", err)
		}
	}(ctx, dp)
}
