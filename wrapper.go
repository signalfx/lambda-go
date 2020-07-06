package sfxlambda

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
)

const (
	name    = "signalfx_lambda_go"
	version = "0.0.1"
)

// HandlerWrapper extends interface lambda.Handler to support sending metric datapoints.
type HandlerWrapper interface {
	Invoke(ctx context.Context, payload []byte) ([]byte, error)
	SendDatapoints(dps []*datapoint.Datapoint) error
}

// handlerWrapper is a HandlerWrapper and lambda.Handler implementation.
// handlerWrapper delegates lambda handler function invocation to the embedded lambda.Handler.
type handlerWrapper struct {
	lambda.Handler
	notColdStart bool
	ctx          context.Context
}

// NewHandlerWrapper is a HandlerWrapper creating factory function.
func NewHandlerWrapper(handler lambda.Handler) HandlerWrapper {
	return &handlerWrapper{Handler: handler}
}

// Invoke is handlerWrapper's lambda.Handler implementation that delegates to the Invoke method of the embedded lambda.Handler.
// Invoke creates and sends metrics.
func (hw *handlerWrapper) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	hw.ctx = ctx
	dps := []*datapoint.Datapoint{hw.invocationsDatapoint()}
	if !hw.notColdStart {
		dps = append(dps, hw.coldStartsDatapoint())
		hw.notColdStart = true
	}
	start := time.Now()
	responseBytes, err := hw.Handler.Invoke(ctx, payload)
	dps = append(dps, hw.durationDatapoint(time.Since(start)))
	if err != nil {
		dps = append(dps, hw.errorsDatapoint())
	}
	if err2 := hw.sendDatapoints(ctx, dps); err2 != nil {
		log.Error(err2)
	}
	return responseBytes, err
}

type dimensions map[string]string

// Start takes HandlerWrapper, a lambda.Handler implementation and passes it function lambda.StartHandler
func Start(handler HandlerWrapper) {
	lambda.StartHandler(handler)
}

// SendDatapoints sends custom metric datapoints to SignalFx.
func (hw *handlerWrapper) SendDatapoints(dps []*datapoint.Datapoint) error {
	return hw.sendDatapoints(hw.ctx, dps)
}

func (hw *handlerWrapper) sendDatapoints(ctx context.Context, dps []*datapoint.Datapoint) error {
	if ctx == nil {
		return fmt.Errorf("invalid argument. context is nil")
	}
	var errs []string
	var dims map[string]string
	var err error
	if dims, err = defaultDimensions(ctx); err != nil {
		errs = append(errs, err.Error())
	}
	// Adding dimensions to datapoints with checking for errors. Valid dimensions (dims) and errors (err) possible.
	for _, dp := range dps {
		dp.Dimensions = datapoint.AddMaps(dims, dp.Dimensions)
	}
	if err = sendDatapoints(ctx, dps); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errs, "\n"))
}

// defaultDimensions derives metric dimensions from AWS Lambda ARN. Formats and examples of AWS Lambda ARNs are in the
// AWS Lambda (Lambda) section at https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
func defaultDimensions(ctx context.Context) (map[string]string, error) {
	var lambdaContext *lambdacontext.LambdaContext
	var ok bool

	if lambdaContext, ok = lambdacontext.FromContext(ctx); !ok {
		return nil, fmt.Errorf("failed to get *LambdaContext from %+v", ctx)
	}
	arnSubstrings := strings.Split(lambdaContext.InvokedFunctionArn, ":")
	dims := dimensions{
		"aws_function_version":     lambdacontext.FunctionVersion,
		"aws_function_name":        lambdacontext.FunctionName,
		"metric_source":            "lambda_wrapper",
		"function_wrapper_version": name + "_" + version,
	}
	var errs []string
	if err := dims.addArnDerivedDimension("aws_region", arnSubstrings, 3); err != nil {
		errs = append(errs, err.Error())
	}
	if err := dims.addArnDerivedDimension("aws_account_id", arnSubstrings, 4); err != nil {
		errs = append(errs, err.Error())
	}
	if len(arnSubstrings) > 5 {
		switch arnSubstrings[5] {
		case "function":
			lambdaArn := ""
			switch len(arnSubstrings) {
			case 8:
				dims["aws_function_qualifier"] = arnSubstrings[7]
				lambdaArn = strings.Join(append(arnSubstrings[:7], lambdacontext.FunctionVersion), ":")
			case 7:
				lambdaArn = strings.Join(append(arnSubstrings, lambdacontext.FunctionVersion), ":")
			}
			if err := dims.addArnDerivedDimension("lambda_arn", []string{lambdaArn}, 0); err != nil {
				errs = append(errs, err.Error())
			}
		case "event-source-mappings":
			dims["lambda_arn"] = lambdaContext.InvokedFunctionArn
			if err := dims.addArnDerivedDimension("event_source_mappings", arnSubstrings, 6); err != nil {
				errs = append(errs, err.Error())
			}
		}
	} else {
		errs = append(errs, fmt.Sprintf("invalid arn. got %d substrings instead of 7 or 8 after colon-splitting the arn %s", len(arnSubstrings), lambdaContext.InvokedFunctionArn))
	}
	if os.Getenv("AWS_EXECUTION_ENV") != "" {
		dims["aws_execution_env"] = os.Getenv("AWS_EXECUTION_ENV")
	}
	if len(errs) == 0 {
		return dims, nil
	}
	return dims, fmt.Errorf(strings.Join(errs, "\n"))
}

func (ds dimensions) addArnDerivedDimension(dimension string, arnSubstrings []string, arnSubstringIndex int) error {
	if len(arnSubstrings) > arnSubstringIndex && arnSubstrings[arnSubstringIndex] != "" {
		ds[dimension] = arnSubstrings[arnSubstringIndex]
		return nil
	}
	return fmt.Errorf("invalid arn caused %s dimension value not to be set. got %d substrings instead of 7 or 8 after colon-splitting the arn %s", dimension, len(arnSubstrings), strings.Join(arnSubstrings, ":"))
}

func (hw *handlerWrapper) invocationsDatapoint() *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.invocations", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	return &dp
}

func (hw *handlerWrapper) coldStartsDatapoint() *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.cold_starts", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	return &dp
}

func (hw *handlerWrapper) durationDatapoint(elapsed time.Duration) *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.duration", Value: datapoint.NewIntValue(Milliseconds(elapsed)), MetricType: datapoint.Gauge}
	return &dp
}

func (hw *handlerWrapper) errorsDatapoint() *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.errors", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	return &dp
}

// Milliseconds returns the duration as an integer millisecond count.
// Added to time.Duration in go 1.13
func Milliseconds(d time.Duration) int64 { return int64(d) / 1e6 }
