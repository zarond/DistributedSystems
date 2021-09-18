FROM golang:latest

ADD ./worker.go /go/src/worker.go
ADD go.mod /go/src/
ADD go.sum /go/src/

WORKDIR /go/src/

#RUN go get .
RUN go build -o /opt/worker /go/src/
#RUN go build -race -o /opt/worker /go/src/

WORKDIR /opt
RUN ls -la

# exec is used so app can listen to signals
CMD ["sh", "-c", "exec /opt/worker $FLAGS"]