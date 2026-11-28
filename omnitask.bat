@echo off
rem OmniTask 引导式工具箱（Windows）
rem 用法：双击 omnitask.bat 或在 cmd 中运行，按提示输入选项数字
setlocal
cd /d "%~dp0"
set "ROOT=%cd%"
set "BUILD=%ROOT%\build"
set "WEB=%ROOT%\backend\internal\server\webdist"

:menu
echo.
echo ================= OmniTask 工具箱 =================
echo  1) 部署（构建前端+后端+Host，迁移，启动）[Windows]
echo  2) 构建后端 server（选择目标平台）
echo  3) 构建脚本 Host（选择目标平台）
echo  4) 构建前端（dist，可内嵌进后端）
echo  5) 构建全部（前端+server+Host，选择目标平台）
echo  6) 自检（后端 build/vet/test + 前端构建 + SDK 测试）
echo  0) 退出
echo =================================================
set "choice="
set /p "choice=请输入选项数字: "
if "%choice%"=="1" ( call :deploy & goto menu )
if "%choice%"=="2" ( call :ask_target & call :build_go server %TARGET% & goto menu )
if "%choice%"=="3" ( call :ask_target & call :build_go host %TARGET% & goto menu )
if "%choice%"=="4" ( call :build_frontend & goto menu )
if "%choice%"=="5" ( call :ask_target & call :build_all %TARGET% & goto menu )
if "%choice%"=="6" ( call :check & goto menu )
if "%choice%"=="0" ( echo 再见 & exit /b 0 )
echo 无效选项，请重新输入。
goto menu

:ask_target
set "TARGET=linux"
echo 选择目标平台：
echo   1) linux-amd64
echo   2) windows-amd64
echo   3) 两者都编译
set "t="
set /p "t=输入 1/2/3 [默认 1]: "
if "%t%"=="2" set "TARGET=windows"
if "%t%"=="3" set "TARGET=both"
exit /b

rem build_go <server|host> <linux|windows|both>
:build_go
set "GOCMD=%~1"
set "GT=%~2"
mkdir "%BUILD%" 2>nul
pushd "%ROOT%\backend"
set "CGO_ENABLED=0"
set "GOARCH=amd64"
if "%GT%"=="linux" goto bg_linux
if "%GT%"=="windows" goto bg_windows
goto bg_linux
:bg_linux
set "GOOS=linux"
echo ==^> 编译 omnitask-%GOCMD% (linux/amd64)
go build -trimpath -ldflags="-s -w" -o "%BUILD%\omnitask-%GOCMD%-linux-amd64" ".\cmd\%GOCMD%"
if not "%GT%"=="both" goto bg_done
:bg_windows
set "GOOS=windows"
echo ==^> 编译 omnitask-%GOCMD% (windows/amd64)
go build -trimpath -ldflags="-s -w" -o "%BUILD%\omnitask-%GOCMD%-windows-amd64.exe" ".\cmd\%GOCMD%"
:bg_done
popd
exit /b

:build_frontend
set "FBASE=%VITE_BASE%"
if "%FBASE%"=="" set "FBASE=/"
echo ==^> 构建前端 (base=%FBASE%)
pushd "%ROOT%\frontend"
set "VITE_BASE=%FBASE%"
call npm run build
if not exist "%WEB%" mkdir "%WEB%"
for %%F in ("%WEB%\*") do if /i not "%%~nxF"==".gitkeep" del /q "%%F" 2>nul
for /d %%D in ("%WEB%\*") do rmdir /s /q "%%D" 2>nul
xcopy /e /i /y "dist\*" "%WEB%\" >nul
popd
exit /b

:build_all
call :build_frontend
call :build_go server %1
call :build_go host %1
echo 构建产物目录：%BUILD%
dir /b "%BUILD%"
exit /b

:check
echo ==^> 后端 build/vet/test
pushd "%ROOT%\backend"
go build ./...
go vet ./...
go test ./...
popd
echo ==^> 前端 build
pushd "%ROOT%\frontend"
call npm run build
popd
echo ==^> SDK 测试
pushd "%ROOT%\sdk\python"
.venv\Scripts\python -m unittest discover -s tests -t .
popd
echo ==^> 全部检查通过
exit /b

:deploy
call :build_frontend
call :build_go server windows
call :build_go host windows
echo ==^> 迁移（lite 自动跳过，full 执行）
"%BUILD%\omnitask-server-windows-amd64.exe" -migrate -conf config.yaml
echo 启动 backend（首次运行会自动生成 config.yaml，lite 会自动拉起 Host）...
start "OmniTask backend" "%BUILD%\omnitask-server-windows-amd64.exe" -conf config.yaml
echo 已在新窗口启动。浏览器打开 http://127.0.0.1:8080
exit /b
