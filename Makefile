.PHONY: build dev run clean

build:
	cd frontend && npm run build
	cd backend && go build -o calendar-app .

run: build
	cd backend && ./calendar-app

dev-backend:
	cd backend && go run main.go

dev-frontend:
	cd frontend && npm run dev

clean:
	rm -f backend/calendar-app
	rm -rf backend/static
