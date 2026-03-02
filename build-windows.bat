@echo off
REM Build the frontend and create a Windows executable.
REM Run this from the project root (e.g. calendar-app\).
setlocal

cd /d "%~dp0"
if not exist frontend (
    echo Error: frontend directory not found. Run from project root.
    exit /b 1
)
if not exist backend (
    echo Error: backend directory not found. Run from project root.
    exit /b 1
)

echo Building frontend...
cd frontend
call npm run build
if errorlevel 1 (
    echo Error: Frontend build failed.
    exit /b 1
)

echo Building Windows executable...
cd ..\backend
set GOOS=windows
set GOARCH=amd64
go build -o calendar-app.exe .
if errorlevel 1 (
    echo Error: Go build failed.
    exit /b 1
)

echo.
echo Done. Output: backend\calendar-app.exe
exit /b 0
