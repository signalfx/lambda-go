# SignalFx Go Lambda Wrapper

SignalFx Golang Lambda Wrapper.

## Usage

The SignalFx Go Lambda Wrapper is a wrapper around an AWS Lambda Go function handler, used to instrument execution of the function and send metrics to SignalFx.

### Installation
To install run the command:

`$ go get https://github.com/signalfx/lambda-go`

### Environment Variable
Set the SIGNALFX_AUTH_TOKEN environment variable with the appropriate SignalFx authentication token. Change the default 
values of the other variables accordingly if desired.

`SIGNALFX_AUTH_TOKEN=<SignalFx authentication token>`

`SIGNALFX_INGEST_ENDPOINT=https://ingest.signalfx.com/v2/datapoint`

`SIGNALFX_SEND_TIMEOUT_SECONDS=5`

###  Wrapping a function
The SignalFx Go Lambda Wrapper wraps a valid Lambda handler function. Calling the `WrappedHandlerFunc()` 
HandlerFuncWrapper method returns the wrapped handler function which can then be passed to the 
`Start()` method. See the example below.

```
import (
  ...
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/signalfx/lambda-go"
  ...
)
...

func handler(...) ... {
  ...  
}
...

func main() {
  ...	
  handlerFuncWrapper := sfxlambda.NewHandlerFuncWrapper(handler)
  wrappedHandlerFunc := handlerFuncWrapper.WrappedHandlerFunc()
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
| aws_execution_env  | AWS execution environment (e.g. AWS_Lambda_go1.x) |
| function_wrapper_version  | SignalFx function wrapper qualifier (e.g. signalfx-lambda-0.0.5) |
| metric_source | The literal value of 'lambda_wrapper' |


### Sending custom metric in the Lambda function
Use the `SendDatapoint()` method of HandlerFuncWrapper to send custom metrics/datapoints to SignalFx from within the 
Lambda handler function. The HandlerFuncWrapper variable needs to be declared globally in order to be accessible within
the Lambda handler function. See example below.

```
import (
  ...
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/signalfx/lambda-go"
  ...
)
...

var handlerFuncWrapper sfxlambda.HandlerFuncWrapper
...

func handler(...) ... {
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
  handlerFuncWrapper.SendDatapoints(ctx, []*datapoint.Datapoint{&dp})
  ...
}
...

func main() {
  ...	
  handlerFuncWrapper = sfxlambda.NewHandlerFuncWrapper(handler)
  wrappedHandlerFunc := handlerFuncWrapper.WrappedHandlerFunc()
  lambda.Start(wrappedHandlerFunc)
  ...
}
...
```


### Testing locally.
Run the command below in the lambda-go package folder

`$ SIGNALFX_AUTH_TOKEN=test go test -v`

## License

Apache Software License v2. Copyright © 2014-2018 SignalFx