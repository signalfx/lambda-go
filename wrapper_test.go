package sfxlambda

import (
	"bytes"
	"context"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

var ctx = lambdacontext.NewContext(context.TODO(), &lambdacontext.LambdaContext{InvokedFunctionArn: "arn:aws:lambda:us-east-1:accountId:function:functionName:$LATEST"})

func TestValidHandlerFunctions(t *testing.T) {
	savedSendDatapoint := sendDatapoint
	sendDatapoint = func(ctx context.Context, dp *datapoint.Datapoint) {}
	defer func() {
		sendDatapoint = savedSendDatapoint
	}()
	var tests = []struct {
		handlerFunc interface{}
		want        error
	}{
		{func() {}, nil},
		{func() error { return nil }, nil},
		{func(interface{}) error { return nil }, nil},
		{func() (interface{}, error) { return nil, nil }, nil},
		{func(interface{}) (interface{}, error) { return nil, nil }, nil},
		{func(context.Context) error { return nil }, nil},
		{func(context.Context, interface{}) error { return nil }, nil},
		{func(context.Context) (interface{}, error) { return nil, nil }, nil},
		{func(context.Context, interface{}) (interface{}, error) { return nil, nil }, nil},
	}
	for _, test := range tests {
		handlerFuncWrapper := NewHandlerFuncWrapper(test.handlerFunc)
		wrappedHandlerFunc := handlerFuncWrapper.WrappedHandlerFunc()
		if _, got := wrappedHandlerFunc(ctx, nil); got != test.want {
			t.Errorf("EXPECTED %+v but GOT %+v", test.want, got)
		}
	}
}

func TestInValidHandlerFunctions(t *testing.T) {
	savedSendDatapoint := sendDatapoint
	sendDatapoint = func(ctx context.Context, dp *datapoint.Datapoint) {}
	defer func() {
		sendDatapoint = savedSendDatapoint
	}()
	var tests = []struct {
		handlerFunc interface{}
	}{
		{func(interface{}, interface{}) {}},
		{func(context.Context, interface{}, interface{}) {}},
		{func() (interface{}, interface{}) { return nil, nil }},
	}
	for _, test := range tests {
		handlerFuncWrapper := NewHandlerFuncWrapper(test.handlerFunc)
		wrappedHandlerFunc := handlerFuncWrapper.WrappedHandlerFunc()
		if _, got := wrappedHandlerFunc(ctx, nil); got == nil {
			t.Errorf("EXPECTED an error but GOT %+v", got)
		}
	}
}

func TestSendDatapoint(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()
	NewHandlerFuncWrapper(func() {}).WrappedHandlerFunc()(ctx, nil)
	time.Sleep(1000 * time.Millisecond)
	if buf.Len() == 0 {
		t.Errorf("EXPECTED error sending datapoint to SignalFx log message")
	}
}
