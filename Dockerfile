FROM golang:alpine

WORKDIR /reconciliation-system

COPY go.mod go.sum ./

RUN go mod download

COPY . .
