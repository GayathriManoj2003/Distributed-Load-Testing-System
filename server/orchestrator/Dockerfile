# Use the official Golang image as a base image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /orchestrator

# Copy only the necessary go.mod and go.sum files to leverage caching
COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the Go application
RUN go build -o orchestrator .

# Run the compiled binary
CMD ["./orchestrator"]