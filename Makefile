NAME=api_server

build:
	go build -o ${NAME} -tags "sqlite_foreign_keys"

linux:
	env CGO_ENABLED=1 CC_FOR_TARGET=x86_64-unknown-linux-gnu-gcc GOOS=linux GOARCH=amd64 CC=x86_64-unknown-linux-gnu-gcc go build -o ${NAME}_linux -tags "sqlite_foreign_keys linux"

run:
	go build -o ${NAME} -tags "sqlite_foreign_keys"
	./${NAME}

clean:
	go clean
	rm ${NAME}
	rm ${NAME}_linux
