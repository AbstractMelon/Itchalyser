@echo off
setlocal EnableDelayedExpansion

REM Function to build for a specific OS and architecture
goto build

REM Usage:
REM build.bat <os> <arch>
REM Example: build.bat linux amd64

if "%1" == "all" goto build-all
if "%1" == "" goto usage
if "%2" == "" goto usage

call :build %1 %2
goto end

REM Build for all
(build-all)
call :build windows amd64
call :build linux amd64
call :build android arm64
goto end

REM Build for a specific OS and architecture
(build)
set output=..\builds\%1\Itchalyser
mkdir "%output%\.."
echo Building for %1-%2
cd src
set GOOS=%1
set GOARCH=%2
go build -o "%output%" main.go
echo Built "%output%"
cd ..
goto end

REM Usage
(usage)
echo Usage: build.bat ^<os^> ^<arch^>
echo Example: build.bat linux amd64
exit /b 1

REM End
(end)

