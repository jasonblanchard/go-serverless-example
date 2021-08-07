pulumi:
	go build -o ./bin/pulumi ./cmd/pulumi

provision: pulumi
	pulumi up

apilambda:
	export GO111MODULE=on
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/apilambda cmd/apilambda/main.go
	zip -j ./bin/apilambda.zip ./bin/apilambda