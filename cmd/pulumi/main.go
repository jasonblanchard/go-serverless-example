package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/codedeploy"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		stack := ctx.Stack()
		caller, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}

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

		lambdaSourceBucket, err := s3.NewBucket(ctx, fmt.Sprintf("go-serverless-example-%s", stack), &s3.BucketArgs{
			Acl: pulumi.String("private"),
		})
		if err != nil {
			return err
		}

		ctx.Export("lambdaSourceBucket", lambdaSourceBucket.Bucket)

		// This will eventually be overwritten by CD pipeline
		// Presumes the project has been built
		initialLambdaBuild, err := s3.NewBucketObject(ctx, "examplebucketObject", &s3.BucketObjectArgs{
			Key:    pulumi.String("initial"),
			Bucket: lambdaSourceBucket.ID(),
			Source: pulumi.NewFileAsset("./bin/apilambda.zip"),
		})
		if err != nil {
			return err
		}

		apilambdafn, err := lambda.NewFunction(ctx, fmt.Sprintf("go-serverless-example-api-%s", stack), &lambda.FunctionArgs{
			Handler:  pulumi.String("apilambda"),
			Role:     role.Arn,
			Runtime:  pulumi.String("go1.x"),
			S3Bucket: lambdaSourceBucket.ID(),
			S3Key:    initialLambdaBuild.Key,
			Publish:  pulumi.BoolPtr(true),
		},
			pulumi.DependsOn([]pulumi.Resource{logPolicy}),
		)
		if err != nil {
			return err
		}

		ctx.Export("apiLambdaName", apilambdafn.Name)

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
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/*/*", "us-east-1", caller.AccountId, apigw.ID()),
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

		// CodeDeploy
		releaseBucket, err := s3.NewBucket(ctx, fmt.Sprintf("go-serverless-example-release-%s", stack), &s3.BucketArgs{
			Acl: pulumi.String("private"),
		})
		if err != nil {
			return err
		}

		ctx.Export("apiLambdaReleaseBucket", releaseBucket.Bucket)

		codeDeployApplication, err := codedeploy.NewApplication(ctx, "go-serverless-example", &codedeploy.ApplicationArgs{
			Name:            pulumi.String("go-serverless-example"),
			ComputePlatform: pulumi.String("Lambda"),
		})
		if err != nil {
			return err
		}

		codeDeployRole, err := iam.NewRole(ctx, "go-serverless-example-codedeploy", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.Any(fmt.Sprintf("%v%v%v%v%v%v%v%v%v%v%v%v%v", "{\n", "  \"Version\": \"2012-10-17\",\n", "  \"Statement\": [\n", "    {\n", "      \"Sid\": \"\",\n", "      \"Effect\": \"Allow\",\n", "      \"Principal\": {\n", "        \"Service\": \"codedeploy.amazonaws.com\"\n", "      },\n", "      \"Action\": \"sts:AssumeRole\"\n", "    }\n", "  ]\n", "}\n")),
		})
		if err != nil {
			return err
		}
		_, err = iam.NewRolePolicyAttachment(ctx, "go-serverless-example-codedeploy-lambda", &iam.RolePolicyAttachmentArgs{
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSCodeDeployRoleForLambda"),
			Role:      codeDeployRole.Name,
		})
		if err != nil {
			return err
		}

		// TODO: Scope this down
		_, err = iam.NewRolePolicyAttachment(ctx, "go-serverless-example-codedeploy-s3-full", &iam.RolePolicyAttachmentArgs{
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonS3FullAccess"),
			Role:      codeDeployRole.Name,
		})
		if err != nil {
			return err
		}

		_, err = codedeploy.NewDeploymentGroup(ctx, "go-serverless-example-codedeploy", &codedeploy.DeploymentGroupArgs{
			AppName:              codeDeployApplication.Name,
			DeploymentGroupName:  pulumi.String("release"),
			DeploymentConfigName: pulumi.String("CodeDeployDefault.LambdaAllAtOnce"),
			ServiceRoleArn:       codeDeployRole.Arn,
			DeploymentStyle: &codedeploy.DeploymentGroupDeploymentStyleArgs{
				DeploymentOption: pulumi.String("WITH_TRAFFIC_CONTROL"),
				DeploymentType:   pulumi.String("BLUE_GREEN"),
			},
		})

		if err != nil {
			return err
		}

		deployspecBucket, err := s3.NewBucket(ctx, fmt.Sprintf("go-serverless-example-deployspec-%s", stack), &s3.BucketArgs{
			Acl: pulumi.String("private"),
			Versioning: &s3.BucketVersioningArgs{
				Enabled: pulumi.Bool(true),
			},
		})

		ctx.Export("apiLambdaDeployspecBucket", deployspecBucket.Bucket)

		return nil
	})
}
