@echo off
REM 构建前端并将文件复制到后端目录的脚本

echo 开始构建前端...

REM 进入前端目录
cd /d "%~dp0config-management\frontend"

REM 安装依赖
call npm install

REM 构建前端
call npm run build

echo 前端构建完成，文件位于 dist 目录

REM 返回项目根目录
cd /d "%~dp0"

REM 检查后端目录是否有dist目录，如果有则删除
if exist ".\config-management\backend\cmd\api\dist" (
    rmdir /s /q ".\config-management\backend\cmd\api\dist"
)

REM 复制前端构建结果到后端目录
xcopy /E /I ".\config-management\frontend\dist" ".\config-management\backend\cmd\api\dist"

echo 前端文件已复制到后端目录
echo 现在可以构建后端了: go build .\config-management\backend\cmd\api\