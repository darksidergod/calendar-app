.PHONY: build build-windows dev run clean

# Build frontend static files and embed them into the Go binary
build:
	cd frontend && npm run build
	cd backend && go build -o calendar-app .

# Build a Windows executable (run from Linux/macOS)
build-windows:
	cd frontend && npm run build
	cd backend && GOOS=windows GOARCH=amd64 go build -o calendar-app.exe .

run: build
	cd backend && ./calendar-app

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

clean:
	rm -f backend/calendar-app backend/calendar-app.exe
	rm -rf backend/static
