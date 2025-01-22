FROM golang:latest

RUN adduser frontend
USER frontend
WORKDIR /home/frontend/app
COPY --chown=frontend:frontend . .
RUN go mod tidy
RUN go build -o ./app
EXPOSE 4443
CMD ["./app"]