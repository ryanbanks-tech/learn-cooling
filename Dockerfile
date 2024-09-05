# Build React app
FROM node:14 AS build
WORKDIR /app
COPY ./web .
RUN npm install && npm run build

# Build Go server
FROM golang:1.22
WORKDIR /app
COPY --from=build /app/build ./build
COPY . .
RUN go build -o learn-cooling ./cmd/learn-cooling

# Run the server
CMD ["./learn-cooling"]
