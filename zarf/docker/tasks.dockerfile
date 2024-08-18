FROM golang:1.23.0 AS build

# disable cgo 
ENV CGO_ENABLED=0

# define args that can be passed when building image
ARG BUILD 

# since we aredoing vendoring, copy everything into /service 
COPY . /service 

WORKDIR /service/app/api/

# build tasks bin 
RUN go build -ldflags "-X main.build =${BUILD}" 



# run bin inside Alpine
FROM alpine:3.20.2

# create a new system group for services and daemons and create a system user that only allowed to run services 
# for reducing security risks inside of the container .
RUN addgroup -g 1000 -S tasks && adduser -u 1000 -h /service -G tasks -S tasks 

# copy bin 
COPY --from=build --chown=tasks:tasks /service/app/api/api /service/api 

WORKDIR /service

USER tasks

CMD [ "./api" ]



