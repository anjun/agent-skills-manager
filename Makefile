# agent-skills-manager (asm)
#
# Usage:
#   make release                 # 自动 patch 升版 → 总结 → 打 tag → push（触发 CI 发版）
#   make release BUMP=minor      # minor / major / patch（默认 patch）
#   make release VERSION=0.3.2   # 显式指定版本
#   make summary / make tags / make test / make build

.PHONY: help test build summary tags release check-clean

VERSION ?=
BUMP    ?= patch

help:
	@echo "Targets:"
	@echo "  make release                 自动升版(默认 patch) + 打 tag + push → CI 发版"
	@echo "  make release BUMP=minor      升 minor / major / patch"
	@echo "  make release VERSION=x.y.z   指定版本（不自动 bump）"
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

check-clean:
	@# 忽略历史遗留的临时文件
	@rm -f .release-version
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "错误: 工作区不干净，请先 commit 或 stash" >&2; \
		git status -sb; \
		exit 1; \
	fi
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "main" ] && [ "$$branch" != "master" ]; then \
		echo "警告: 当前分支是 $$branch（通常应在 main 上打 tag）" >&2; \
	fi

# 一键：干净检查 → 解析版本（不写盘）→ 总结 → 打 tag → push → CI Release
release: check-clean
	@set -e; \
	rm -f .release-version; \
	if [ -n "$(VERSION)" ]; then \
		RAW="$(VERSION)"; \
		RAW="$${RAW#v}"; \
		RAW="$${RAW#V}"; \
	else \
		BASE="$(LAST_TAG)"; \
		if [ -z "$$BASE" ]; then BASE="v0.0.0"; fi; \
		BASE="$${BASE#v}"; \
		maj=$${BASE%%.*}; rest=$${BASE#*.}; \
		min=$${rest%%.*}; pat=$${rest#*.}; \
		pat=$${pat%%-*}; \
		case "$(BUMP)" in \
			major) maj=$$((maj+1)); min=0; pat=0 ;; \
			minor) min=$$((min+1)); pat=0 ;; \
			patch|"") pat=$$((pat+1)) ;; \
			*) echo "错误: BUMP 应为 major|minor|patch，收到: $(BUMP)" >&2; exit 2 ;; \
		esac; \
		RAW="$$maj.$$min.$$pat"; \
		echo "未指定 VERSION，按 BUMP=$(BUMP) 自动升版: $${BASE:-0.0.0} → $$RAW"; \
	fi; \
	echo "$$RAW" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$$' || { \
		echo "错误: 版本格式应为 x.y.z，得到: $$RAW" >&2; exit 2; \
	}; \
	TAG="v$$RAW"; \
	echo "→ 将发布: $$TAG"; \
	echo ""; \
	echo "=========================================="; \
	echo "  分支: $$(git rev-parse --abbrev-ref HEAD)"; \
	echo "  HEAD: $$(git rev-parse --short HEAD)"; \
	echo "  上一正式 tag: $(or $(LAST_TAG),（无）)"; \
	echo "=========================================="; \
	if [ -n "$(LAST_TAG)" ]; then \
		echo ""; \
		echo "变更范围: $(LAST_TAG)..HEAD"; \
		echo "------------------------------------------"; \
		git log --oneline --no-decorate "$(LAST_TAG)..HEAD" || true; \
		echo "------------------------------------------"; \
		git diff --stat "$(LAST_TAG)..HEAD" || true; \
		if [ -z "$$(git log $(LAST_TAG)..HEAD --oneline 2>/dev/null)" ]; then \
			echo "警告: 自上一正式 tag 以来没有新 commit" >&2; \
		fi; \
	fi; \
	if git rev-parse "$$TAG" >/dev/null 2>&1; then \
		echo "错误: tag $$TAG 已存在" >&2; \
		exit 1; \
	fi; \
	echo ""; \
	echo "将创建并推送: $$TAG"; \
	printf "确认? [y/N] "; read ans; \
	case "$$ans" in y|Y|yes|YES) ;; *) echo "已取消"; exit 1 ;; esac; \
	git tag -a "$$TAG" -m "Release $$TAG"; \
	echo "→ 推送分支 $$(git rev-parse --abbrev-ref HEAD) …"; \
	git push origin "$$(git rev-parse --abbrev-ref HEAD)"; \
	echo "→ 推送 tag $$TAG …"; \
	git push origin "$$TAG"; \
	echo ""; \
	echo "完成。GitHub Actions Release 将构建并上传二进制:"; \
	repo=$$(git remote get-url origin | sed -E 's#.*github.com[:/](.+)$$#\1#' | sed 's/\.git$$//'); \
	echo "  https://github.com/$$repo/actions"; \
	echo "  https://github.com/$$repo/releases/tag/$$TAG"; \
	echo "  make 不负责本地编译 dist/；发版由 CI 完成。"
