# SignalFx Go Lambda Wrapper

SignalFx Golang Lambda Wrapper.

## Usage

The SignalFx Go Lambda Wrapper is a wrapper around an AWS Lambda Go function handler, used to instrument execution of the function and send metrics to SignalFx.

### Install via maven dependency
To install 

`$ go get https://github.com/signalfx/lambda-go`

### Environment Variable
Set the required environment variable below:

`SIGNALFX_AUTH_TOKEN=<signalfx token>`

Optionally the environment variables below can be set:

`SIGNALFX_INGEST_ENDPOINT=<ingest endpoint, default=https://ingest.signalfx.com/v2/datapoint>`

`SIGNALFX_SEND_TIMEOUT=<timeout in seconds for sending datapoint, default=5>`

###  Wrapping a function
The SignalFx Go Lambda Wrapper wraps a valid lambda handler function upon the creation of a HandlerFuncWrapper object. Calling 
the GetWrappedHandlerFunc() HandlerFuncWrapper method returns the wrapped handler function which can then be passed to the 
Start() method. See the example below.

The example below includes sending a custom metric in lambda handler function. If nil is passed as the context argument 
for method SendDatapoint() then SendDatapoint() the background context context.Background() by default.

```
import (
  ...
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/signalfx/lambda-go"
  ...
)

var handlerFuncWrapper sfxlambda.HandlerFuncWrapper
...

func handler(...) (...) {
  ...  
  // Timeout context for sending custom metric.
  ctx, _ := context.WithTimeout(context.Background(), 200 * time.Millisecond)
  // Custom counter metric.
  dp := datapoint.Datapoint {
      Metric: "db_calls",
      Value: datapoint.NewIntValue(1),
      MetricType: datapoint.Counter,
      Dimensions: map[string]string{"db_name":"mysql1",},
  }
  // Sending custom metric to SignalFx.
  handlerFuncWrapper.SendDatapoint(ctx, &dp)
  ...
}
...

func main() {
  ...	
  handlerFuncWrapper = sfxlambda.NewHandlerFuncWrapper(handler)
  wrappedHandlerFunc := handlerFuncWrapper.GetWrappedHandlerFunc()
  lambda.Start(wrappedHandlerFunc)
  ...
}
...
```

### Metrics and dimensions sent by the wrapper

The Lambda wrapper sends the following metrics to SignalFx:

| Metric Name  | Type | Description |
| ------------- | ------------- | ---|
| function.invocations  | Counter  | Count number of Lambda invocations|
| function.cold_starts  | Counter  | Count number of cold starts|
| function.errors  | Counter  | Count number of errors from underlying Lambda handler|
| function.duration  | Gauge  | Milliseconds in execution time of underlying Lambda handler|

The Lambda wrapper adds the following dimensions to all data points sent to SignalFx:

| Dimension | Description |
| ------------- | ---|
| lambda_arn  | ARN of the Lambda function instance |
| aws_region  | AWS Region  |
| aws_account_id | AWS Account ID  |
| aws_function_name  | AWS Function Name |
| aws_function_version  | AWS Function Version |
| aws_function_qualifier  | AWS Function Version Qualifier (version or version alias if it is not an event source mapping Lambda invocation) |
| event_source_mappings  | AWS Function Name (if it is an event source mapping Lambda invocation) |
| aws_execution_env  | AWS execution environment (e.g. AWS_Lambda_java8) |
| function_wrapper_version  | SignalFx function wrapper qualifier (e.g. signalfx-lambda-0.0.5) |
| metric_source | The literal value of 'lambda_wrapper' |

### Testing locally.
Run the command below in the lambda-go package folder

`$ SIGNALFX_AUTH_TOKEN=test go test -v`

## License

Apache Software License v2. Copyright © 2014-2017 SignalFx