# agent-skills-manager (asm)
#
# Usage:
#   make release VERSION=0.3.2   # 总结变更 → 打 tag → push（触发 GitHub Actions 发版）
#   make release VERSION=v0.3.2
#   make summary                # 仅打印相对上一 tag 的变更摘要
#   make tags                   # 列出本地 tag
#   make test / make build

.PHONY: help test build summary tags release check-version check-clean

VERSION ?=
TAG     = v$(patsubst v%,%,$(VERSION))

help:
	@echo "Targets:"
	@echo "  make release VERSION=x.y.z   总结 git 变更、打 tag、push（触发 CI 发版）"
	@echo "  make summary                 相对上一正式 tag 的 commit 摘要"
	@echo "  make tags                    列出 tag"
	@echo "  make test / make build"

test:
	go test ./...

build:
	@mkdir -p bin
	go build -ldflags "-s -w -X main.version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
		-o bin/agent-skills-manager ./cmd/agent-skills-manager
	ln -sfn agent-skills-manager bin/asm
	@echo "→ bin/asm  $$("./bin/asm" version)"

# 上一正式版 tag（忽略 -ci / pre 等带连字符的）
LAST_TAG = $(shell git tag -l 'v*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -1)

summary:
	@echo "=========================================="
	@echo "  分支: $$(git rev-parse --abbrev-ref HEAD)"
	@echo "  HEAD: $$(git rev-parse --short HEAD)"
	@echo "  上一正式 tag: $(or $(LAST_TAG),（无）)"
	@echo "=========================================="
	@if [ -n "$(LAST_TAG)" ]; then \
		echo ""; \
		echo "变更范围: $(LAST_TAG)..HEAD"; \
		echo "------------------------------------------"; \
		git log --oneline --no-decorate "$(LAST_TAG)..HEAD" || true; \
		echo "------------------------------------------"; \
		echo "统计:"; \
		git diff --stat "$(LAST_TAG)..HEAD" || true; \
	else \
		echo ""; \
		echo "全部提交:"; \
		git log --oneline --no-decorate | head -30; \
	fi
	@echo ""
	@echo "工作区状态:"
	@git status -sb

tags:
	@git tag -l 'v*' --sort=-v:refname | head -20

check-version:
	@if [ -z "$(VERSION)" ]; then \
		echo "错误: 请指定 VERSION，例如: make release VERSION=0.3.2" >&2; \
		exit 2; \
	fi
	@echo "$(VERSION)" | grep -Eq '^[vV]?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || { \
		echo "错误: VERSION 格式应为 x.y.z 或 vX.Y.Z（可选 pre 后缀）" >&2; \
		exit 2; \
	}

check-clean:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "错误: 工作区不干净，请先 commit 或 stash" >&2; \
		git status -sb; \
		exit 1; \
	fi
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "main" ] && [ "$$branch" != "master" ]; then \
		echo "警告: 当前分支是 $$branch（通常应在 main 上打 tag）" >&2; \
	fi

# 一键：总结 → 打 tag → push tag（及当前分支）→ CI Release 自动构建上传
release: check-version check-clean summary
	@if git rev-parse "$(TAG)" >/dev/null 2>&1; then \
		echo "错误: tag $(TAG) 已存在" >&2; \
		exit 1; \
	fi
	@echo ""
	@echo "将创建并推送: $(TAG)"
	@printf "确认? [y/N] "; read ans; \
	case "$$ans" in y|Y|yes|YES) ;; *) echo "已取消"; exit 1 ;; esac
	@git tag -a "$(TAG)" -m "Release $(TAG)"
	@echo "→ 推送分支 $$(git rev-parse --abbrev-ref HEAD) …"
	@git push origin "$$(git rev-parse --abbrev-ref HEAD)"
	@echo "→ 推送 tag $(TAG) …"
	@git push origin "$(TAG)"
	@echo ""
	@echo "完成。GitHub Actions Release 将构建并上传二进制:"
	@echo "  https://github.com/$$(git remote get-url origin | sed -E 's#.*github.com[:/](.+)(\.git)?#\1#' | sed 's/\.git$$//')/actions"
	@echo "  make 不负责本地编译 dist/；发版由 CI 完成。"
