DISCIT_DIR = /home/shared/projects/discit

.PHONY: up down health logs discit-up discit-down

up: discit-up
	docker compose up -d --build
	@# discit servislerini omega-net'e bağla
	docker network connect omega-net discit-integrations-api 2>/dev/null || true
	docker network connect omega-net discit-demand-app 2>/dev/null || true

down:
	docker compose down
	docker network disconnect omega-net discit-integrations-api 2>/dev/null || true
	docker network disconnect omega-net discit-demand-app 2>/dev/null || true

discit-up:
	docker network create omega-net 2>/dev/null || true
	cd $(DISCIT_DIR) && docker compose up -d

discit-down:
	cd $(DISCIT_DIR) && docker compose down

health:
	@echo "=== Platforms ==="
	@curl -sf http://localhost:20000/health | python3 -m json.tool 2>/dev/null || echo "DOWN"
	@echo "=== Discit Demand ==="
	@curl -sf http://localhost:8020/health | python3 -m json.tool 2>/dev/null || echo "DOWN"
	@echo "=== Discit Supply Tools ==="
	@curl -sf http://localhost:8011/health | python3 -m json.tool 2>/dev/null || echo "DOWN"

logs:
	docker compose logs -f
