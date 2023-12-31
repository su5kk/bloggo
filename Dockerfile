# Use a specific version for reproducibility
FROM golang:1.21

# Set the working directory
WORKDIR /app

# Copy only go.mod and go.sum for dependency caching
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod and go.sum haven't changed)
RUN go mod download

# Copy the rest of the source code
COPY . .

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o bloggo

# Command to run the executable
CMD ["./bloggo"]
