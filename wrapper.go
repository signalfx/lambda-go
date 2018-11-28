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
	GetWrappedHandlerFunc() func(context.Context, interface{}) (interface{}, error)
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
		if hfw.defaultDimensions, err = getDefaultDimensions(ctx); err == nil {
			hfw.sendInvocationsDatapoint()
			hfw.sendColdStartsDatapoint()
			var payloadBytes, responseBytes []byte
			if payloadBytes, err = json.Marshal(payload); err == nil {
				if responseBytes, err = lambda.NewHandler(handlerFunc).Invoke(ctx, payloadBytes); err == nil {
					if nonErrorReturnType := getNonErrorReturnType(handlerFunc); nonErrorReturnType != nil {
						response = reflect.New(nonErrorReturnType).Interface()
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

// GetWrappedHandlerFunc returns the wrapped lambda handler function.
func (hfw *handlerFuncWrapper) GetWrappedHandlerFunc() func(context.Context, interface{}) (interface{}, error) {
	return hfw.wrappedHandlerFunc
}

// SendDatapoint sends custom metrics to SignalFx. If ctx is nil the background context is used.
func (hfw *handlerFuncWrapper) SendDatapoint(ctx context.Context, dp *datapoint.Datapoint) {
	sendDatapoint(ctx, dp)
}

func getDefaultDimensions(ctx context.Context) (map[string]string, error) {
	if lambdaContext, ok := lambdacontext.FromContext(ctx); ok {
		if strings.TrimSpace(lambdaContext.InvokedFunctionArn) == "" {
			return nil, fmt.Errorf("lambda function arn cannot be blank")
		}
		// Expected function arn format arn:aws:lambda:us-east-1:accountId:function:functionName:$LATEST
		arnTokens := strings.Split(lambdaContext.InvokedFunctionArn, ":")
		dimensions := map[string]string{
			"aws_function_version": lambdacontext.FunctionVersion,
			"aws_function_name":    lambdacontext.FunctionName,
			"aws_region":           arnTokens[3],
			"aws_account_id":       arnTokens[4],
			"metric_source":        "lambda_wrapper",
			//'function_wrapper_version': name + '_' + version,
		}
		if os.Getenv("AWS_EXECUTION_ENV") != "" {
			dimensions["ws_execution_env"] = os.Getenv("AWS_EXECUTION_ENV")
		}
		if arnTokens[5] == "function" {
			if len(arnTokens) == 8 {
				dimensions["aws_function_qualifier"] = arnTokens[7]
				arnTokens[7] = lambdacontext.FunctionVersion
			} else if len(arnTokens) == 7 {
				arnTokens = append(arnTokens, lambdacontext.FunctionVersion)
			}
			dimensions["lambda_arn"] = strings.Join(arnTokens, ":")
		} else if arnTokens[5] == "event-source-mappings" {
			dimensions["event_source_mappings"] = arnTokens[6]
			dimensions["lambda_arn"] = lambdaContext.InvokedFunctionArn
		}
		return dimensions, nil
	}
	return nil, fmt.Errorf("failed to get *LambdaContext from %+v", ctx)
}

func getNonErrorReturnType(handlerFunc interface{}) reflect.Type {
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
