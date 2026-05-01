.PHONY: help dev dev-backend dev-frontend build build-backend build-frontend tidy clean

help:
	@echo "Moon Panel — make targets:"
	@echo "  make dev            run backend (:3000) and frontend (:5173) together"
	@echo "  make dev-backend    run only the Go API"
	@echo "  make dev-frontend   run only the Vite dev server"
	@echo "  make build          build a single-binary release into ./dist"
	@echo "  make tidy           go mod tidy + npm install"
	@echo "  make clean          remove build artifacts"

# Dev backend gets MOON_CORS_ORIGINS set explicitly so the Vite dev server
# (different origin :5173) can talk to it. Production same-origin deploys
# leave MOON_CORS_ORIGINS empty in the image's default env.
DEV_CORS=http://localhost:5173

dev:
	@echo "Starting backend on :3000 and frontend on :5173 ..."
	@(cd backend && MOON_CORS_ORIGINS=$(DEV_CORS) go run ./cmd/server) & \
	 (cd frontend && npm run dev) ; \
	 wait

dev-backend:
	cd backend && MOON_CORS_ORIGINS=$(DEV_CORS) go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

build-frontend:
	cd frontend && npm run build

build-backend:
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ../dist/moon-panel ./cmd/server

build: build-frontend build-backend
	@echo "Built: ./dist/moon-panel"

tidy:
	cd backend && go mod tidy
	cd frontend && npm install

clean:
	rm -rf dist
	rm -rf backend/web/dist/*
	rm -rf frontend/dist
	rm -rf frontend/node_modules
