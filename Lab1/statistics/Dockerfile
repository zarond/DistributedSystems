FROM golang:latest

ADD ./stats.go /go/src/stats.go
ADD go.mod /go/src/
ADD go.sum /go/src/

WORKDIR /go/src/

#RUN go get .
RUN go build -o /opt/stats /go/src/
#RUN go build -race -o /opt/worker /go/src/

WORKDIR /opt
RUN ls -la

# exec is used so app can listen to signals
CMD ["sh", "-c", "exec /opt/stats $FLAGS"]