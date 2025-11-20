FROM golang:1.21-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o worker main.go

FROM scratch

WORKDIR /

COPY --from=build /app/worker /worker

CMD ["/worker"]