package sfxlambda

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"testing"
)

var ctx = lambdacontext.NewContext(context.TODO(), &lambdacontext.LambdaContext{InvokedFunctionArn: "arn:aws:lambda:us-east-1:accountId:function:functionName:$LATEST"})

func TestValidHandlerFunctions(t *testing.T) {
	var tests = []struct {
		handlerFunc interface{}
	}{
		{func() {}},
		{func() error { return nil }},
		{func(interface{}) error { return nil }},
		{func() (interface{}, error) { return nil, nil }},
		{func(interface{}) (interface{}, error) { return nil, nil }},
		{func(context.Context) error { return nil }},
		{func(context.Context, interface{}) error { return nil }},
		{func(context.Context) (interface{}, error) { return nil, nil }},
		{func(context.Context, interface{}) (interface{}, error) { return nil, nil }},
	}
	for _, test := range tests {
		input, _ := json.Marshal("")
		if _, err := (&HandlerWrapper{Handler: lambda.NewHandler(test.handlerFunc)}).Invoke(ctx, input); err != nil {
			t.Errorf("valid lambda handler function invocation error. got %+v", err)
		}
	}
}

func TestInValidHandlerFunctions(t *testing.T) {
	var tests = []struct {
		handlerFunc interface{}
	}{
		{func(interface{}, interface{}) {}},
		{func(context.Context, interface{}, interface{}) {}},
		{func() (interface{}, interface{}) { return nil, nil }},
	}
	for _, test := range tests {
		input, _ := json.Marshal("")
		if _, err := (&HandlerWrapper{Handler: lambda.NewHandler(test.handlerFunc)}).Invoke(ctx, input); err == nil {
			t.Errorf("invalid lambda handler function invocation error. got %+v", err)
		}
	}
}

const (
	arKey = "aws_region"
	acKey = "aws_account_id"
	afKey = "aws_function_qualifier"
	laKey = "lambda_arn"
	esKey = "event_source_mappings"
)

// Testing default metric dimensions derived from AWS Lambda ARN.
// AWS Lambda ARN syntax/examples used as input:
// arn:aws:lambda:region:account-id:function:function-name
// arn:aws:lambda:region:account-id:function:function-name:alias-name
// arn:aws:lambda:region:account-id:function:function-name:version
// arn:aws:lambda:region:account-id:event-source-mappings:event-source-mapping-id
func TestDefaultDimensions(t *testing.T) {
	var tests = []struct {
		arn             string
		functionVersion string
		want            map[string]string
	}{
		{"",
			"version",
			map[string]string{
				arKey: "",
				acKey: "",
				afKey: "",
				laKey: "",
				esKey: ""},
		},
		{"arn:aws:lambda:region:account-id:function:function-name",
			"version",
			map[string]string{
				arKey: "region",
				acKey: "account-id",
				afKey: "",
				laKey: "arn:aws:lambda:region:account-id:function:function-name:version",
				esKey: ""},
		},
		{"arn:aws:lambda:region:account-id:function:function-name:alias-name",
			"version",
			map[string]string{
				arKey: "region",
				acKey: "account-id",
				afKey: "alias-name",
				laKey: "arn:aws:lambda:region:account-id:function:function-name:version",
				esKey: ""},
		},
		{"arn:aws:lambda:region:account-id:function:function-name:version",
			"version",
			map[string]string{
				arKey: "region",
				acKey: "account-id",
				afKey: "version",
				laKey: "arn:aws:lambda:region:account-id:function:function-name:version",
				esKey: ""},
		},
		{"arn:aws:lambda:region:account-id:event-source-mappings:event-source-mapping-id",
			"version",
			map[string]string{
				arKey: "region",
				acKey: "account-id",
				afKey: "",
				laKey: "arn:aws:lambda:region:account-id:event-source-mappings:event-source-mapping-id",
				esKey: "event-source-mapping-id"},
		},
	}
	savedFunctionVersion := lambdacontext.FunctionVersion
	defer func() {
		lambdacontext.FunctionVersion = savedFunctionVersion
	}()
	for _, test := range tests {
		lambdacontext.FunctionVersion = test.functionVersion
		got, _ := defaultDimensions(newCtx(test.arn))
		keys := []string{arKey, acKey, afKey, laKey, esKey}
		for _, k := range keys {
			if got[k] != test.want[k] {
				t.Errorf("want %s got %s", test.want[k], got[k])
			}
		}
	}
}

func newCtx(arn string) context.Context {
	return lambdacontext.NewContext(context.TODO(), &lambdacontext.LambdaContext{InvokedFunctionArn: arn})
}
