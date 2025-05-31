FROM golang:alpine

WORKDIR /reconciliation_system

COPY go.mod go.sum ./

RUN go mod download

COPY . .
