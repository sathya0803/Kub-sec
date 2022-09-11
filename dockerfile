 FROM golang:1.16-buster AS build
 WORKDIR /app
 COPY . ./
 RUN go mod download
 RUN go build -o /main
 
 ##
 ## Deploy
 ##
 FROM gcr.io/distroless/base-debian10
 WORKDIR /
 COPY --from=build /main /main
 EXPOSE 8080
 ENTRYPOINT ["/main"]