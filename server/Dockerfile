FROM golang:1.18.6-alpine3.15
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go mod download
RUN go build -o out .
EXPOSE 8181
CMD ["/app/out"]