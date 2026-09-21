.PHONY: build test run-server run-agent

# 构建后端与脚本 Host 到仓库 bin/
build:
	go build -trimpath -ldflags "-s -w" -o ../bin/omnitask-server ./cmd/server
	go build -trimpath -ldflags "-s -w" -o ../bin/omnitask-agent ./cmd/agent

# 自检（与 omnitask.sh check 的后端部分一致）
test:
	go build ./...
	go vet ./...
	go test ./...

run-server:
	go run ./cmd/server -conf config.yaml

run-agent:
	go run ./cmd/agent -conf config.yaml
