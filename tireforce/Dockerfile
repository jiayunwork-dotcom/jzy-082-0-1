# 构建阶段：Go 1.22
FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -o /bin/tireforce ./cmd/server

# 运行阶段：Go 1.22 基础镜像
FROM golang:1.22
COPY --from=build /bin/tireforce /usr/local/bin/tireforce
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["tireforce"]
