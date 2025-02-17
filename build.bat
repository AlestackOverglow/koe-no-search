@echo off
set GOPATH=C:\Users\Alesta\go
"C:\Program Files\Go\bin\go.exe" build -v -ldflags "-X 'filesearch/internal/search.Version=0.2.1' -H windowsgui" -o koe-no-search-gui.exe ./cmd/gui 