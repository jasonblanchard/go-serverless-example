.PHONY: apilambda

GIT_SHA=$(shell git rev-parse HEAD)

pulumi:
	go build -o ./bin/pulumi ./cmd/pulumi

provision: pulumi
	pulumi up

apilambda:
	export GO111MODULE=on
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/apilambda cmd/apilambda/main.go
	zip -j ./bin/apilambda.zip ./bin/apilambda

apipush:
	aws s3 cp ./bin/apilambda.zip s3://$$(pulumi stack output lambdaSourceBucket)/${GIT_SHA}/apilambda.zip

lambdaversion:
	aws lambda update-function-code --function-name $$(pulumi stack output apiLambdaName) --s3-bucket $$(pulumi stack output lambdaSourceBucket) --s3-key ${GIT_SHA}/apilambda.zip --publish

release:
	aws s3 cp ./release.yaml s3://$$(pulumi stack output apiLambdaReleaseBucket)/release.yaml

deployspec:
	zip -j ./deployspec.zip ./deployspec.yaml
	aws s3 cp ./deployspec.zip s3://$$(pulumi stack output apiLambdaDeployspecBucket)

sha:
	echo ${GIT_SHA}