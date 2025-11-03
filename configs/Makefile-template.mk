# ——— Configuration —————————————————————————————————————————————

COMPOSE_FILE    ?= compose.yaml
SSH_TARGET      := $(TOPO_TARGET)
DOCKER_CONTEXT  := $(SSH_TARGET)

PROJECT := $(shell awk -F': *' '/^name:/ {print $$2; exit}' $(COMPOSE_FILE))

# ——— Targets ———————————————————————————————————————————————————

.PHONY: all check-remote check-docker build create-context transfer up

all: check-remote check-docker build create-context transfer up

# 1️⃣ Ensure the remote host is reachable

check-remote:
	@echo "🔌 Checking remote host availability..."
	@ssh -o BatchMode=yes -o ConnectTimeout=5 $(SSH_TARGET) exit || (echo "❌ Remote host $(SSH_TARGET) unreachable"; exit 1)

# 2️⃣ Verify Docker is installed on the remote

check-docker:
	@echo "🐳 Checking for Docker on remote host..."
	@ssh -o BatchMode=yes -o ConnectTimeout=5 $(SSH_TARGET) docker version > /dev/null 2>&1 || (echo "❌ Docker CLI not found on remote host"; exit 1)

# 3️⃣ Build images locally using the default context

build:
	@echo "🏗 Building images in context 'default'..."
	@echo $(COMPOSE_DIR)
	@echo $(COMPOSE_BASE)
	@docker --context default compose -f $(COMPOSE_FILE) build

# 4️⃣ Create the target Docker context if absent

create-context:
	@echo "🔍 Checking for Docker context '$(DOCKER_CONTEXT)'..."
	@docker context ls --format '{{.Name}}' | grep -Fxq $(DOCKER_CONTEXT) \
		|| (echo "➕ Creating context '$(DOCKER_CONTEXT)'" && \
		docker context create $(DOCKER_CONTEXT) --docker host=ssh://$(DOCKER_CONTEXT))

# 5️⃣ Save & load each image on the remote host

transfer:
	@echo "🚚 Saving & loading images to $(DOCKER_CONTEXT)..."
	@for svc in $$(docker --context default compose \
			-f $(COMPOSE_FILE) config --services); do \
				image="$(PROJECT)-$$svc"; \
			echo "  • $$image → $(DOCKER_CONTEXT)"; \
			docker --context default save "$$image" | docker --context $(DOCKER_CONTEXT) load; \
		done

# 6️⃣ Start services on the remote without rebuilding

up:
	@echo "🚀 Bringing up services on the board..."
	@docker --context $(DOCKER_CONTEXT) compose -f $(COMPOSE_FILE) up -d --no-build --remove-orphans
