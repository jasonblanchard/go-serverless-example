.PHONY: apilambda

pulumi:
	go build -o ./bin/pulumi ./cmd/pulumi

provision: pulumi
	pulumi up

apilambda:
	export GO111MODULE=on
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/apilambda cmd/apilambda/main.go
	zip -j ./bin/apilambda.zip ./bin/apilambda

apipush:
	aws s3 cp ./bin/apilambda.zip s3://go-serverless-example-prod-126ca7c/v1/apilambda.zip

lambdaversion:
	aws lambda update-function-code --function-name go-serverless-example-api-prod-0c7d815 --s3-bucket go-serverless-example-prod-126ca7c --s3-key v1/apilambda.zip --publish

release:
	aws s3 cp ./release.yaml s3://go-serverless-example-release-prod-1f070f7/release.yaml