FROM golang:1.18.3 as build
WORKDIR /app
ADD . .
RUN make all

FROM alpine:3.15.4  as runtime

COPY --from=build /app/app .
COPY --from=build /app/config.ini .

RUN apk --no-cache add ca-certificates wget
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
RUN wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.28-r0/glibc-2.28-r0.apk
RUN apk add glibc-2.28-r0.apk

ENTRYPOINT ./app -file=./config.ini
