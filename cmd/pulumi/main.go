package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		stack := ctx.Stack()

		// API Gateway
		apigw, err := apigatewayv2.NewApi(ctx, fmt.Sprintf("go-serverless-example-%s", stack), &apigatewayv2.ApiArgs{
			ProtocolType: pulumi.String("HTTP"),
		})
		if err != nil {
			return err
		}

		_, err = apigatewayv2.NewStage(ctx, "stagev2", &apigatewayv2.StageArgs{
			ApiId:      apigw.ID(),
			Name:       pulumi.String("$default"),
			AutoDeploy: pulumi.Bool(true),
		})

		if err != nil {
			return err
		}

		// API lambda
		role, err := iam.NewRole(ctx, "task-exec-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Sid": "",
					"Effect": "Allow",
					"Principal": {
						"Service": "lambda.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		logPolicy, err := iam.NewRolePolicy(ctx, "lambda-log-policy", &iam.RolePolicyArgs{
			Role: role.Name,
			Policy: pulumi.String(`{
                "Version": "2012-10-17",
                "Statement": [{
                    "Effect": "Allow",
                    "Action": [
                        "logs:CreateLogGroup",
                        "logs:CreateLogStream",
                        "logs:PutLogEvents"
                    ],
                    "Resource": "arn:aws:logs:*:*:*"
                }]
            }`),
		})

		apilambdafn, err := lambda.NewFunction(ctx, fmt.Sprintf("go-serverless-example-api-%s", stack), &lambda.FunctionArgs{
			Handler: pulumi.String("apilambda"),
			Role:    role.Arn,
			Runtime: pulumi.String("go1.x"),
			Code:    pulumi.NewFileArchive("./bin/apilambda.zip"),
			// S3Bucket: pulumi.String("serverles-go-test-again-serverlessdeploymentbuck-1m2tl9fx70cxp"),
			// S3Key:    pulumi.String("noop"),
			Publish: pulumi.BoolPtr(true),
		},
			pulumi.DependsOn([]pulumi.Resource{logPolicy}),
		)
		if err != nil {
			return err
		}

		apilambdaReleaseAlias, err := lambda.NewAlias(ctx, "releaseLambdaAlias", &lambda.AliasArgs{
			Name:            pulumi.String("release"),
			FunctionName:    apilambdafn.Name,
			FunctionVersion: pulumi.String("$LATEST"),
		})
		if err != nil {
			return err
		}

		// API Gateway => API Lambda integration

		_, err = lambda.NewPermission(ctx, "APIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  apilambdafn.Name,
			Qualifier: apilambdaReleaseAlias.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/*/*", "us-east-1", "076797644834", apigw.ID()), // TODO: Parameterize account ID
		})
		if err != nil {
			return err
		}

		apilambdaIntegration, err := apigatewayv2.NewIntegration(ctx, "apilambda-integration", &apigatewayv2.IntegrationArgs{
			ApiId:                apigw.ID(),
			IntegrationType:      pulumi.String("AWS_PROXY"),
			IntegrationUri:       apilambdaReleaseAlias.InvokeArn,
			IntegrationMethod:    pulumi.String("POST"),
			PayloadFormatVersion: pulumi.String("1.0"),
		})
		if err != nil {
			return err
		}

		target := apilambdaIntegration.ID().OutputState.ApplyT(func(id pulumi.ID) string {
			return fmt.Sprintf("integrations/%s", id)
		}).(pulumi.StringOutput)

		_, err = apigatewayv2.NewRoute(ctx, "routev2", &apigatewayv2.RouteArgs{
			ApiId:    apigw.ID(),
			RouteKey: pulumi.String("GET /{proxy+}"),
			Target:   target,
			// AuthorizerId:      authorizer.ID(),
			// AuthorizationType: pulumi.String("JWT"),
		})
		if err != nil {
			return err
		}

		return nil
	})
}
