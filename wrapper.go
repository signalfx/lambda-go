package sfxlambda

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/signalfx/golib/datapoint"
	"os"
	"reflect"
	"strings"
	"time"
)

// HandlerFuncWrapper is the interface that wraps the lambda handler function.
type HandlerFuncWrapper interface {
	WrappedHandlerFunc() func(context.Context, interface{}) (interface{}, error)
	SendDatapoint(context.Context, *datapoint.Datapoint)
}

type handlerFuncWrapper struct {
	wrappedHandlerFunc func(context.Context, interface{}) (interface{}, error)
	defaultDimensions  map[string]string
	notColdStart       bool
}

// NewHandlerFuncWrapper is the HandlerFuncWrapper factory function.
func NewHandlerFuncWrapper(handlerFunc interface{}) HandlerFuncWrapper {
	hfw := handlerFuncWrapper{}
	hfw.wrappedHandlerFunc = func(ctx context.Context, payload interface{}) (interface{}, error) {
		var response interface{}
		var err error
		start := time.Now()
		if hfw.defaultDimensions, err = defaultDimensions(ctx); err == nil {
			hfw.sendInvocationsDatapoint()
			hfw.sendColdStartsDatapoint()
			var payloadBytes, responseBytes []byte
			if payloadBytes, err = json.Marshal(payload); err == nil {
				if responseBytes, err = lambda.NewHandler(handlerFunc).Invoke(ctx, payloadBytes); err == nil {
					if returnType := nonErrorReturnType(handlerFunc); returnType != nil {
						response = reflect.New(returnType).Interface()
						err = json.Unmarshal(responseBytes, &response)
					}
				}
			}
		}
		if err != nil {
			hfw.sendErrorsDatapoint()
		}
		hfw.sendDurationDatapoint(time.Since(start))
		return response, err
	}
	return &hfw
}

// WrappedHandlerFunc returns the wrapped lambda handler function.
func (hfw *handlerFuncWrapper) WrappedHandlerFunc() func(context.Context, interface{}) (interface{}, error) {
	return hfw.wrappedHandlerFunc
}

// SendDatapoint sends custom metrics to SignalFx. If ctx is nil the background context is used.
func (hfw *handlerFuncWrapper) SendDatapoint(ctx context.Context, dp *datapoint.Datapoint) {
	sendDatapoint(ctx, dp)
}

func defaultDimensions(ctx context.Context) (map[string]string, error) {
	var lambdaContext *lambdacontext.LambdaContext
	var ok bool
	if lambdaContext, ok = lambdacontext.FromContext(ctx); !ok {
		return nil, fmt.Errorf("failed to get *LambdaContext from %+v", ctx)
	}
	arnTokens := strings.Split(lambdaContext.InvokedFunctionArn, ":")
	dimensions := map[string]string{
		"aws_function_version": lambdacontext.FunctionVersion,
		"aws_function_name":    lambdacontext.FunctionName,
		"metric_source":        "lambda_wrapper",
		//'function_wrapper_version': name + '_' + version,
	}
	switch {
	case len(arnTokens) > 3: dimensions["aws_region"] = arnTokens[3]
	case len(arnTokens) > 4: dimensions["aws_account_id"] = arnTokens[4]
	case len(arnTokens) > 5:
		switch arnTokens[5] {
		case "function":
			switch len(arnTokens) {
			case 8:
				dimensions["aws_function_qualifier"] = arnTokens[7]
				arnTokens[7] = lambdacontext.FunctionVersion
			case 7:
				arnTokens = append(arnTokens, lambdacontext.FunctionVersion)
			}
			dimensions["lambda_arn"] = strings.Join(arnTokens, ":")
		case "event-source-mappings":
			if len(arnTokens) > 6 {
				dimensions["event_source_mappings"] = arnTokens[6]
			}
			dimensions["lambda_arn"] = lambdaContext.InvokedFunctionArn
		}
	}
	if os.Getenv("AWS_EXECUTION_ENV") != "" {
		dimensions["ws_execution_env"] = os.Getenv("AWS_EXECUTION_ENV")
	}
	return dimensions, nil
}

func nonErrorReturnType(handlerFunc interface{}) reflect.Type {
	handlerFuncType := reflect.TypeOf(handlerFunc)
	if handlerFuncType.NumOut() == 2 {
		return handlerFuncType.Out(0)
	}
	return nil
}

func (hfw *handlerFuncWrapper) sendInvocationsDatapoint() {
	dp := datapoint.Datapoint{Metric: "function.invocations", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	dp.Dimensions = datapoint.AddMaps(hfw.defaultDimensions, dp.Dimensions)
	sendDatapoint(nil, &dp)
}

func (hfw *handlerFuncWrapper) sendColdStartsDatapoint() {
	if !hfw.notColdStart {
		dp := datapoint.Datapoint{Metric: "function.cold_starts", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
		dp.Dimensions = datapoint.AddMaps(hfw.defaultDimensions, dp.Dimensions)
		sendDatapoint(nil, &dp)
	}
	hfw.notColdStart = true
}

func (hfw *handlerFuncWrapper) sendDurationDatapoint(elapsed time.Duration) {
	dp := datapoint.Datapoint{Metric: "function.duration", Value: datapoint.NewFloatValue(elapsed.Seconds()), MetricType: datapoint.Gauge}
	dp.Dimensions = datapoint.AddMaps(hfw.defaultDimensions, dp.Dimensions)
	sendDatapoint(nil, &dp)
}

func (hfw *handlerFuncWrapper) sendErrorsDatapoint() {
	dp := datapoint.Datapoint{Metric: "function.errors", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	dp.Dimensions = datapoint.AddMaps(hfw.defaultDimensions, dp.Dimensions)
	sendDatapoint(nil, &dp)
}
