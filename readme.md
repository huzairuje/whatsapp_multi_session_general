## Overview
---
This project is for engine whatsapp multi-session using library https://github.com/tulir/whatsmeow to use emulate whatsapp web.

### prerequisite
a. gcc (dev essential libs on linux) or using mingw on windows platform
b. golang version >= 1.20

### build up
a. windows platform
1. install mingw or gcc from trusted sources like choco or another package manager
2. build on windows on this command
   ```shell
       set GOOS=windows && set GOARCH=amd64 && set CGO_ENABLED=1 && go build -o bin/whatsapp_multi_session-windows-amd64.exe
   ```
b. linux or unix platform
1. install gcc or using `sudo apt install build-essential`
2. build on linux or unix to target linux based on your server
   ```shell
       env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o bin/whatsapp_multi_session-linux-amd64
   ```
   or to another platform such as windows
   ```shell
       env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o bin/whatsapp_multi_session-windows-amd64.exe
   ```
    
