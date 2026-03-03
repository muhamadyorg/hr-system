FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY client/ ./client/
COPY shared/ ./shared/
COPY attached_assets/ ./attached_assets/
COPY vite.config.ts tsconfig.json tailwind.config.ts postcss.config.js components.json ./
RUN npx vite build

FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY goserver/ ./goserver/
RUN go build -o hr-system main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/hr-system ./hr-system
COPY --from=frontend-builder /app/dist/public ./dist/public
EXPOSE 5000
ENV PORT=5000
CMD ["./hr-system"]
