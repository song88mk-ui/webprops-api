FROM golang:latest AS build

WORKDIR /app

ADD go.mod /app
ADD go.sum /app

RUN go mod download

COPY . /app/
RUN CGO_ENABLED=0 go build -o bin/webprops main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app/bin/webprops /app/bin/webprops
ENV HOME=/app
WORKDIR /app

ENV PORT=8000
EXPOSE $PORT

ENTRYPOINT ["/app/bin/webprops"]
