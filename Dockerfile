FROM golang:1.24 as compiler

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mandok main.go
# Use a minimal image for the final build
# to reduce size and attack surface
# Use the Docker CLI image to run the binary

FROM docker:cli

WORKDIR /usr/src/app
ENV PORT=80
COPY --from=compiler /usr/src/app/mandok ./mandok
VOLUME /usr/src/app/projects
CMD ["./mandok"]