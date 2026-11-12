# TODO(Haoyu): Maintain these shared entry points; commands run from the repository root.
GO ?= go
PYTHON ?= python3
PNPM ?= pnpm
COMPOSE = docker compose -f deployments/compose.yaml
CONTROLLERS ?= http://127.0.0.1:8001,http://127.0.0.1:8002,http://127.0.0.1:8003
MANIFEST ?= .data/fixture/manifest.json
INTERNAL_CONTROLLERS = http://controller-1:8000,http://controller-2:8000,http://controller-3:8000

.PHONY: setup build backend frontend fmt vet test test-race test-integration smoke test-raft test-e2e test-media test-media-browser test-player up down fixture media ingest demo experiment-baseline experiment-main plot
setup:
	$(GO) mod download
	cd frontend && $(PNPM) install --frozen-lockfile
build: backend frontend
backend:
	$(GO) build -o bin/controller ./cmd/controller
	$(GO) build -o bin/peer ./cmd/peer
	$(GO) build -o bin/client ./cmd/client
	$(GO) build -o bin/demo-api ./cmd/demo-api
frontend:
	cd frontend && $(PNPM) run build
fmt:
	$(GO) fmt ./...
vet:
	$(GO) vet ./...
test:
	$(GO) test ./...
test-race:
	$(GO) test -race ./...
test-integration:
	$(GO) test ./tests/integration/...
smoke: backend
	$(PYTHON) tests/system/smoke.py
test-media: backend
	$(PYTHON) tests/system/media.py
test-media-browser: backend frontend
	BROWSER_TEST=1 $(PYTHON) tests/system/media.py
test-player:
	cd frontend && $(PNPM) test
test-raft: smoke
test-e2e: smoke
up:
	$(COMPOSE) up --build -d
down:
	$(COMPOSE) down
fixture:
	$(PYTHON) scripts/media/fixture.py
media:
	bash scripts/media/generate_fixture.sh
ingest:
	$(COMPOSE) run --rm --no-deps ingest $(MANIFEST) --controllers $(INTERNAL_CONTROLLERS)
demo:
	@echo "Dashboard: http://localhost:8080 (start services and ingest a video first)"
experiment-baseline:
	$(COMPOSE) run --rm --no-deps experiment-runner --scenario experiments/scenarios/baseline.json --controllers $(INTERNAL_CONTROLLERS)
experiment-main:
	$(COMPOSE) run --rm --no-deps experiment-runner --scenario experiments/scenarios/main.json --controllers $(INTERNAL_CONTROLLERS)
plot:
	$(PYTHON) experiments/aggregate.py results/baseline/sessions.jsonl
	$(PYTHON) experiments/plot.py results/baseline/sessions.jsonl
