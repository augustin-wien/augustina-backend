FROM golang:1.20
WORKDIR /app
COPY ./app .
RUN go mod download
CMD ["go run app.go"]
