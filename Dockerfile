FROM golang:1.20
WORKDIR /app
COPY ./app .
RUN go mod download
RUN go install github.com/jackc/tern/v2@latest
CMD ["tern migrate && go run main.go"]
