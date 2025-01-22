FROM golang:latest

RUN adduser backend
USER backend
WORKDIR /home/backend/app
COPY --chown=backend:backend . .
RUN go mod tidy
RUN go build -o ./app
EXPOSE 8080
CMD ["./app"]