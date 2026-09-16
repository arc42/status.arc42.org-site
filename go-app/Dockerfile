# syntax=docker/dockerfile:1

FROM golang:1.21

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./
COPY internal/ ./internal/


# Build. GIT_COMMIT is passed by the deploy (see .github/workflows/fly.yml and
# the fly-deploy target in the Makefile) and ends up in the table's footer.
ARG GIT_COMMIT=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.gitCommit=${GIT_COMMIT}" -o /arc42-stats

# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can (optionally) document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8043

# Run
CMD [ "/arc42-stats" ]
