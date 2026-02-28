@echo off
REM Build the frontend and create a Windows executable.
REM Run this from the project root (e.g. calendar-app\).
cd frontend
call npm run build
cd ..\backend
set GOOS=windows
set GOARCH=amd64
go build -o calendar-app.exe .
echo.
echo Done. Output: backend\calendar-app.exe
