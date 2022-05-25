#!/usr/bin/env pwsh

go build -o tg-auth-proxy.exe
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

./tg-auth-proxy.exe
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}