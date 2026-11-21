.PHONY: build run clean lint lint-new fmt-check tools

# 优先用 PATH 里的 golangci-lint，没有则用项目内 .tools/ 下的本地副本
GOLANGCI ?= $(shell command -v golangci-lint 2>/dev/null || echo ./.tools/golangci-lint)

build:
	go build -o bin/collector-service ./cmd/server

run:
	go run ./cmd/server

clean:
	rm -rf bin/

# 全量静态检查（errcheck / staticcheck / govet / errorlint / rowserrcheck / forcetypeassert）
lint:
	$(GOLANGCI) run ./...

# 只报相对 HEAD 的增量问题——存量代码多、新增代码严，日常开发用这个
lint-new:
	$(GOLANGCI) run ./... --new-from-rev=HEAD

# 仅检查格式，不修改文件（只列出未格式化文件；输出为空即合规）
# 注：不能用 git ls-files，重构期索引里存有已删除的旧路径，gofmt 会因文件不存在而失败
fmt-check:
	@find . -name '*.go' -not -path './.tools/*' -not -path './vendor/*' -exec gofmt -l {} +

# 安装检查工具到项目内 .tools/（已被 gitignore）
tools:
	GOBIN=$$PWD/.tools go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
