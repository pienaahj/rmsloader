# Start from base image
# FROM golang:alpine as builder
# Add the target architecture
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder
# Print platform info
RUN echo "Building for platform: $TARGETPLATFORM"
# Set build arguments (passed from docker-compose)
ARG TARGETPLATFORM
ARG BUILDPLATFORM  
ARG TARGETARCH  # Docker buildx automatically provides this based on TARGETPLATFORM 

# Define environment variables dynamically
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GO111MODULE=on 

# Set the current working directory inside the container
WORKDIR /go/src/app

# Copy go mod and sum files
COPY ./backend/go.mod .
# - this is only needed if there are dependancies
COPY ./backend/go.sum . 

# Download all dependencies
RUN go mod download

# Copy source from current directory to working directory
COPY ./backend/ .

# Build the application - called run
RUN go build -o ./run .

# Stage 2: Final stage using Ubuntu
FROM ubuntu:24.04

# Update package lists and install necessary packages for Ubuntu
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata curl

# Set the timezone
ENV TZ="Africa/Johannesburg"
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Set the working directory in the final image
WORKDIR /root/

# Copy necessary files from the builder stage
COPY --from=builder /go/src/app/pathConfig.json .
# COPY --from=builder /go/src/app/logs ./logs
COPY --from=builder /go/src/app/keys/rsa_private_dev.pem ./keys/rsa_private_dev.pem
COPY --from=builder /go/src/app/keys/rsa_public_dev.pem ./keys/rsa_public_dev.pem
COPY --from=builder /go/src/app/run .

# Copy the csv files and recording files to the final image
# ensure directory exist
RUN mkdir -p ./recordings
RUN mkdir -p ./recordings/csv

# COPY data/csv/ ./recordings/csv/

# CMD ["./run"]
CMD ["sh", "-c", "echo '🟡 Container started'; ls -l /root; echo '🟢 Trying to exec ./run'; ./run; echo '🔴 run exited with code $?'; sleep 60"]
# CMD ["sh", "-c", "ls -l && pwd && echo 'Contents of /:' && ls -l / && ./init.sh && reflex -r \\.go$$ -s -- sh -c 'go run ./ ' "]


