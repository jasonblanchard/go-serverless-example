pulumi:
	go build -o ./bin/pulumi ./cmd/pulumi

provision:
	pulumi up