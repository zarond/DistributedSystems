FROM golang:latest

ADD ./gateway.go /go/src/gateway.go
ADD go.mod /go/src/
ADD go.sum /go/src/

WORKDIR /go/src/

#RUN go get .
RUN go build -o /opt/gateway /go/src/
#RUN go build -race -o /opt/gateway /go/src/

WORKDIR /opt
RUN ls -la

# exec is used so app can listen to signals
CMD ["sh", "-c", "exec /opt/gateway $FLAGS"]