# Use an official Go runtime as a parent image
FROM golang:latest

# Install GCC (required for cgo)
RUN apt-get update && apt-get install -y build-essential

# Set the working directory inside the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Download Go dependencies (assumes you are using Go modules)
RUN go mod download

# Compile the Go app
RUN go build -o java-code-parser

# Expose port 8080 to the outside world
EXPOSE 8080

# Run the application when the container launches
CMD ["./java-code-parser"]
