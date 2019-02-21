# SignalFx Go Lambda Wrapper
SignalFx Golang Lambda Wrapper.

## Usage
The SignalFx Go Lambda Wrapper is a wrapper around an AWS Lambda Go function handler, used to instrument execution of the function and send metrics to SignalFx.

### Installation
To install run the command:

`$ go get https://github.com/signalfx/lambda-go`

#### Configuring the ingest endpoint

By default, this function wrapper will send to the `us0` realm. If you are
not in this realm you will need to set the `SIGNALFX_INGEST_ENDPOINT` environment
variable to the correct realm ingest endpoint (https://ingest.{REALM}.signalfx.com/v2/datapoint).
To determine what realm you are in, check your profile page in the SignalFx
web application (click the avatar in the upper right and click My Profile).

### Environment Variable
Set the SIGNALFX_AUTH_TOKEN environment variable with the appropriate SignalFx authentication token. Change the default 
values of the other variables accordingly if desired.

`SIGNALFX_AUTH_TOKEN=<SignalFx authentication token>`

`SIGNALFX_INGEST_ENDPOINT=https://ingest.{REALM}.signalfx.com/v2/datapoint`

`SIGNALFX_SEND_TIMEOUT_SECONDS=5`

###  Wrapping a function
The SignalFx Go Lambda Wrapper wraps the handler `lambda.Handler`. Use the `lambda.NewHandler()` function to create the 
handler by passing your Lambda handler function to `lambda.NewHandler()`. Pass the created handler to the 
`sfxlambda.NewHandlerWrapper` function to create the wrapper `sfxlambda.HandlerWrapper`. Finally, pass the created wrapper 
to the `sfxlambda.Start()` function. See the example below.

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
  handlerWrapper := sfxlambda.NewHandlerWrapper(lambda.NewHandler(handler))
  sfxlambda.Start(handlerWrapper)
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
| function_wrapper_version  | SignalFx function wrapper qualifier (e.g. signalfx_lambda_go-0.0.1) |
| metric_source | The literal value of 'lambda_wrapper' |


### Sending custom metric in the Lambda function
Use the method `sfxlambda.SendDatapoint()` of `HandlerWrapper` to send custom metric datapoints to SignalFx from within your 
Lambda handler function. A `sfxlambda.HandlerWrapper` variable needs to be declared globally in order to be accessible 
from within your Lambda handler function. See example below.

```
import (
  ...
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/signalfx/lambda-go"
  ...
)
...

var handlerWrapper sfxlambda.HandlerWrapper
...

func handler(...) ... {
  ...  
  // Custom counter metric.
  dp := datapoint.Datapoint {
      Metric: "db_calls",
      Value: datapoint.NewIntValue(1),
      MetricType: datapoint.Counter,
      Dimensions: map[string]string{"db_name":"mysql1",},
  }
  // Sending custom metric to SignalFx.
  handlerWrapper.SendDatapoints([]*datapoint.Datapoint{&dp})
  ...
}
...

func main() {
  ...
  handlerWrapper = sfxlambda.NewHandlerWrapper(lambda.NewHandler(handler))
  sfxlambda.Start(handlerWrapper)
  ...
}
...
```

### Testing locally.
Run the command below in the lambda-go package folder

`$ SIGNALFX_AUTH_TOKEN=test go test -v`

## License

Apache Software License v2. Copyright © 2014-2018 SignalFx
