rem for some reasons this block only works directly from cmd
set GOOS=android
set GOARCH=arm
go build -o main_android.arm

set GOOS=linux
set GOARCH=arm
go build -o main_linux.arm

set GOOS=linux
set GOARCH=arm64
go build -o main_linux.arm64

set GOOS=linux
set GOARCH=amd64
go build -o main_linux.amd64

set GOOS=linux
set GOARCH=386
go build -o main_linux.386


set GOOS=windows
set GOARCH=amd64
go build -o main_win_amd64.exe

set GOOS=windows
set GOARCH=386
go build -o main_win_386.exe