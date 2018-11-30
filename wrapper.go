package sfxlambda

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

// HandlerWrapper is a lambda.Handler implementation that delegates to the embedded lambda.Handler.
type HandlerWrapper struct {
	lambda.Handler
	notColdStart bool
}

// Invoke is HandlerWrapper's lambda.Handler implementation that delegates to the Invoke method of the embedded lambda.Handler.
// Invoke creates and sends metrics.
func (hw *HandlerWrapper) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
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
	if err2 := hw.SendDatapoints(ctx, dps); err2 != nil {
		log.Error(err2)
	}
	return responseBytes, err
}

type dimensions map[string]string

// Start takes a handler function and creates a HandlerWrapper which is a lambda.Handler implementation.
// Start then passes the HandlerWrapper to method lambda.StartHandler
func Start(handler interface{}) {
	lambda.StartHandler(&HandlerWrapper{Handler: lambda.NewHandler(handler)})
}

// SendDatapoints sends custom metrics to SignalFx.
func (hw *HandlerWrapper) SendDatapoints(ctx context.Context, dps []*datapoint.Datapoint) error {
	if ctx == nil {
		return fmt.Errorf("invalid argument. context is nil")
	}
	dims, err := defaultDimensions(ctx)
	// Adding dimensions to datapoints with checking for errors. Valid dimensions (dims) and errors (err) possible.
	for _, dp := range dps {
		dp.Dimensions = datapoint.AddMaps(dims, dp.Dimensions)
	}
	sendDatapoints(ctx, dps)
	return err
}

// defaultDimensions derives metric dimensions from AWS Lambda ARN. Formats and examples of AWS Lambda ARNs are in the
// AWS Lambda (Lambda) section at https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
func defaultDimensions(ctx context.Context) (map[string]string, error) {
	var lambdaContext *lambdacontext.LambdaContext
	var ok bool
	var errs []string
	if lambdaContext, ok = lambdacontext.FromContext(ctx); !ok {
		return nil, fmt.Errorf("failed to get *LambdaContext from %+v", ctx)
	}
	arnSubstrings := strings.Split(lambdaContext.InvokedFunctionArn, ":")
	dims := dimensions{
		"aws_function_version": lambdacontext.FunctionVersion,
		"aws_function_name":    lambdacontext.FunctionName,
		"metric_source":        "lambda_wrapper",
		//'function_wrapper_version': name + '_' + version,
	}
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
		dims["ws_execution_env"] = os.Getenv("AWS_EXECUTION_ENV")
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

func (hw *HandlerWrapper) invocationsDatapoint() *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.invocations", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	return &dp
}

func (hw *HandlerWrapper) coldStartsDatapoint() *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.cold_starts", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	return &dp
}

func (hw *HandlerWrapper) durationDatapoint(elapsed time.Duration) *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.duration", Value: datapoint.NewFloatValue(elapsed.Seconds()), MetricType: datapoint.Gauge}
	return &dp
}

func (hw *HandlerWrapper) errorsDatapoint() *datapoint.Datapoint {
	dp := datapoint.Datapoint{Metric: "function.errors", Value: datapoint.NewIntValue(1), MetricType: datapoint.Counter}
	return &dp
}
