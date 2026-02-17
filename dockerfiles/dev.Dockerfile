FROM golang:1.25

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

WORKDIR /go/src/github.com/microcosm-collective/microcosm

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN git config --global --add safe.directory /srv

RUN make build

RUN mkdir /etc/microcosm

RUN if [ -e config/api.conf ]; then \
  cp config/api.conf /etc/microcosm/; \
else \
  cp config/api.conf.example /etc/microcosm/api.conf; \
fi

CMD goose up && ./microcosm
