package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Gin cold start")
	r := gin.Default()
	ginLambda = ginadapter.New(r)

	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/api/meta", func(c *gin.Context) {
		apiGwContext, err := ginLambda.GetAPIGatewayContext(c.Request)
		context := fmt.Sprintf("%+v", apiGwContext)

		apiGwStageVars, err := ginLambda.GetAPIGatewayStageVars(c.Request)
		stageVars := fmt.Sprintf("%+v", apiGwStageVars)

		requestId := apiGwContext.RequestID
		stage := apiGwContext.Stage

		authorizer := apiGwContext.Authorizer

		version := lambdacontext.FunctionVersion

		sha := os.Getenv("sha")

		if err != nil {
			c.JSON(500, err)
			return
		}

		c.JSON(200, gin.H{
			"context":    context,
			"stageVars":  stageVars,
			"requestId":  requestId,
			"stage":      stage,
			"authorizer": authorizer,
			"version":    version,
			"sha":        sha,
		})
	})

	r.GET("/api/me", func(c *gin.Context) {
		apiGwContext, err := ginLambda.GetAPIGatewayContext(c.Request)
		if err != nil {
			c.JSON(500, err)
			return
		}
		user := apiGwContext.Identity.User

		c.JSON(200, gin.H{
			user: user,
		})
	})
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println(fmt.Sprintf("%+v", req))
	return ginLambda.Proxy(req)
}

func main() {
	lambda.Start(Handler)
}
