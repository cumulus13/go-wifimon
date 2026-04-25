@echo off
setlocal

set "APP_NAME=wifimon"
set "OUTPUT=%APP_NAME%.exe"
set "MAIN_PACKAGE=.\cmd\wifimon"

echo Building %APP_NAME%...
go build -buildvcs=false -ldflags="-s -w" -o "%OUTPUT%" "%MAIN_PACKAGE%"
if errorlevel 1 (
    echo.
    echo Build failed.
    exit /b 1
)

echo.
echo Build success: %CD%\%OUTPUT%
endlocal
