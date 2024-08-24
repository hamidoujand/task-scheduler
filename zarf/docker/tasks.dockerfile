# Stage 1: Build the docker-cli
FROM alpine:3.20 AS docker-cli
RUN apk add --no-cache docker-cli

# Stage 2: Build your application
FROM golang:1.23.0 AS build

# Disable cgo 
ENV CGO_ENABLED=0

# Define args that can be passed when building image
ARG BUILD 

# Since we are doing vendoring, copy everything into /service 
COPY . /service 

WORKDIR /service/app/api/

# Build tasks binary 
RUN go build -ldflags "-X main.build=${BUILD}" 

# Stage 3: Create the final image
FROM alpine:3.20

# Copy docker-cli binary from the docker-cli stage
COPY --from=docker-cli /usr/bin/docker /usr/bin/docker

# Copy RSA key used for development only
COPY --from=build /service/zarf/keys/. /service/zarf/keys/.

# Copy binary 
COPY --from=build /service/app/api/api /service/api 

WORKDIR /service

# Run the container as root
CMD [ "./api" ]
