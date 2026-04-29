@echo off
REM 构建前端并将文件复制到后端目录的脚本 (Windows 兼容)

setlocal

echo 开始构建前端...

REM 设置路径
set "SCRIPT_DIR=%~dp0"
REM 去除路径末尾的反斜杠
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
set "FRONTEND_DIR=%SCRIPT_DIR%\config-management\frontend"
set "BACKEND_API_DIR=%SCRIPT_DIR%\config-management\backend\cmd\api"
set "BACKEND_DIST_DIR=%BACKEND_API_DIR%\dist"

REM 进入前端目录
cd /d "%FRONTEND_DIR%"

REM 安装依赖
echo 正在安装前端依赖...
call npm install
if %errorlevel% neq 0 (
    echo npm install 失败
    exit /b %errorlevel%
)

REM 构建前端
echo 正在构建前端...
call npm run build
if %errorlevel% neq 0 (
    echo npm run build 失败
    exit /b %errorlevel%
)

echo 前端构建完成，文件位于 dist 目录

REM 检查后端目录是否有dist目录，如果有则删除
if exist "%BACKEND_DIST_DIR%" (
    echo 清理旧的后端 dist 目录...
    rmdir /s /q "%BACKEND_DIST_DIR%"
)

REM 复制前端构建结果到后端目录
echo 正在复制前端文件到后端目录...
xcopy /E /I /Y "%FRONTEND_DIR%\dist" "%BACKEND_DIST_DIR%"

echo 前端文件已复制到后端目录
echo 现在可以构建后端了: go build .\config-management\backend\cmd\api\

endlocal
#!/bin/bash
# 构建前端并将文件复制到后端目录的脚本 (Linux/macOS 兼容)

set -e

echo "开始构建前端..."

# 获取脚本所在目录的绝对路径，确保在任何位置执行脚本都能正确找到路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="${SCRIPT_DIR}/config-management/frontend"
BACKEND_DIST_DIR="${SCRIPT_DIR}/config-management/backend/cmd/api/dist"

# 进入前端目录
cd "${FRONTEND_DIR}"

# 安装依赖
echo "正在安装前端依赖..."
npm install

# 构建前端
echo "正在构建前端..."
npm run build

echo "前端构建完成，文件位于 dist 目录"

# 检查后端目录是否有dist目录，如果有则删除
if [ -d "${BACKEND_DIST_DIR}" ]; then
    echo "清理旧的后端 dist 目录..."
    rm -rf "${BACKEND_DIST_DIR}"
fi

# 复制前端构建结果到后端目录
echo "正在复制前端文件到后端目录..."
cp -r "${FRONTEND_DIR}/dist" "${BACKEND_DIST_DIR}/.."

echo "前端文件已复制到后端目录"
echo "现在可以构建后端了: go build ./config-management/backend/cmd/api/"