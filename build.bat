@echo off
:: Please use govvv when possible! (go install github.com/JoshuaDoes/govvv@latest)

govvv build -ldflags="-s -w" -o sfbot.exe
:: go build -ldflags="-s -w" -o sfbot.exe
