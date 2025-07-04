# Start from base image
# FROM golang:alpine as builder
# Add the target architecture
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder
# Set the current working directory inside the container
WORKDIR /go/src/app

# Set the module env on
ENV GO111MODULE=on

# Install the reflex package
# RUN go install github.com/cespare/reflex@latest

# Copy go mod and sum files
COPY go.mod .
# - this is only needed if there are dependancies
COPY go.sum . 

# Download all dependencies
RUN go mod download

# Copy source from current directory to working directory
COPY . .

# Copy the initialization script into the container
# COPY ./init.sh .


# Build the application - called run
RUN go build -o ./run .

# # Start with a fresh Alpine image
FROM alpine:latest
# Start with a fresh Ubuntu image needs bash
# FROM ubuntu:latest

# Set the current working directory inside the container
WORKDIR /root/

# Install ca-certificates for TLS support
RUN apk --no-cache add ca-certificates

# Get the timezone db for the container
RUN apk add --no-cache tzdata

# Set the timezone
ENV TZ="Africa/Johannesburg"

# Create a symbolic link from /usr/share/zoneinfo/$TZ to /etc/localtime, and also writes the timezone to /etc/timezone
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone


# Set default value for INITIALIZATION_MODE
ENV INITIALIZATION_MODE=p

# Copy SSH key files into the container for future agginment of the correct keys for assignment
COPY id_rsa /root/.ssh/id_rsa
COPY id_rsa.pub /root/.ssh/id_rsa.pub

# Set appropriate permissions on the SSH key files to root only
RUN chmod 600 /root/.ssh/id_rsa

# Copy executable from builder
COPY --from=builder /go/src/app/run .

# Copy the configuration file
COPY --from=builder /go/src/app/pathConfig.json .

# Copy the initialization script
# COPY --from=builder /go/src/app/init.sh .

# Copy logs from the build stage
COPY --from=builder /go/src/app/logs ./logs

# Set execute permission on the script
# RUN chmod +x init.sh

# Expose port 50051 for gPRC
# EXPOSE 50051

# Start the application using reflex and the init.sh script
CMD ["./run"]
# CMD ["sh", "-c", "./init.sh"]
# CMD ["sh", "-c", "./init.sh && reflex -r \\.go$$ -s -- sh -c 'go run ./ ' "]
# CMD ["sh", "-c", "./init.sh && reflex -r \\.go$$ -s -- sh -c 'go run ./ ' "]
# CMD ["sh", "-c", "./init.sh"]
# CMD ["sh", "-c", "ls -l && pwd && echo 'Contents of /:' && ls -l / && ./init.sh && reflex -r \\.go$$ -s -- sh -c 'go run ./ ' "]


