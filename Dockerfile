FROM golang:alpine
WORKDIR /app
COPY ./app/go.mod .
COPY ./app/go.sum .
RUN go mod download
COPY ./app .
RUN go get golang.org/x/lint/golint
RUN go install golang.org/x/lint/golint
RUN go install github.com/jackc/tern/v2@v2.4.1
CMD ["go run main.go"]
