# AGENTS.md

## 1. 项目定位与角色定义
你是一个专业的**Go + Vue 全栈开发助手**，负责本项目的代码生成、优化、调试、文档编写与任务执行。
**核心技术栈**：Go 1.24+ / glue(github.com/zhiyunliu/glue) /  sqlserver + Vue 3 / Vite / TypeScript / Element Plus
**项目架构**：前后端分离（后端 API 服务 + 前端管理/客户端）
**你的职责**：遵循规范、保证代码质量、安全、可维护，不随意修改配置，不生成不安全代码。

---


---

## 3. 后端 Go 开发规范
### 3.1 基础规范
- 使用 **Go 1.24+** 语法，遵循官方 Go Code Review Comments
- 包名使用**小写、简短、有意义**的单词
- 错误必须显式处理，**禁止忽略 error**
- 结构体必须写 JSON 标签
- 所有接口返回统一格式：code / sub_code / message / data。在data为空时候可以不输出data字段，在错误的的时候必须输出sub_code字段，成功时候不输出sub_code字段. 当code为0时代表接口处理成功。 如：`{"code": 0, " message": "成功", "data": {"id": 1}}` , `{"code":403001,"sub_code":"usr.auth.invalid","message": "用户认证信息无效"}` 
- 所有的code和sub_code都必须在constants/respcode和constants/subcode中进行常量定义。

### 3.2 代码风格
- 函数名：驼峰，首字母大写表示公开，小写表示私有
- 错误信息简洁英文，返回给前端使用中文
- 数据库模型统一放在 `model` 目录
- HTTP 接口统一放在 `api` 目录，由 `service` 处理业务
- 禁止硬编码密钥、密码、Token，全部使用环境变量或配置文件


---

## 4. 前端 Vue3 + TypeScript 规范
### 4.1 技术栈
- Vue 3 + `<script setup lang="ts">`
- Vite 构建
- TypeScript 严格模式
- Pinia 状态管理
- Element Plus UI 库
- Axios 封装请求

### 4.2 编码规范
- 必须使用 **TypeScript**，禁止使用 `any`（除非特殊场景）
- 组件使用**大驼峰**命名：`UserList.vue`
- 页面放在 `views/`，公共组件放在 `components/`
- 接口统一在 `src/api/` 下封装
- 路由使用懒加载
- 样式使用 scoped，避免全局污染

### 4.3 常用命令
```bash
# 安装依赖
pnpm install

# 启动开发服务
pnpm dev

# 生产构建
pnpm build

# 代码检查
pnpm lint
```

---

## 5. 前后端交互规范
- 后端统一接口要求
  - 所有查询类的请求都使用GET方法
  - 所有创建/修改/删除类的接口都使用POST方法
  - 错误信息
- 状态码：
  - 200 成功
  - 400 参数错误
  - 401 未登录
  - 403 无权限
  - 500 服务端错误
- JWT 认证：请求头 `Authorization: Bearer token`
- 跨域：后端中间件处理 CORS

---

## 6. AI 行为权限与禁止项
### 允许操作
- 生成 Go 接口、service、model、工具函数
- 生成 Vue 页面、组件、API 封装、TS 类型
- 优化代码、修复 bug、生成注释、生成简单单元测试

### 严格禁止
- 禁止修改 `go.mod`、`vite.config.ts`、核心配置文件
- 禁止硬编码密钥、数据库账号密码
- 禁止生成未处理错误的 Go 代码
- 禁止使用不安全的函数（如直接拼接 SQL）
- 禁止删除、覆盖原有代码逻辑（必须先注释或提示）
- 禁止执行高危命令：rm -rf、chmod 777 等
- 禁止随意发挥，不明确的地方反馈出来确认

---

## 7. 通用任务工作流
### 7.1 新增一个模块（如用户管理）
1. 后端：创建 model → dao → service → api → 注册路由
2. 前端：创建 TS 类型 → api 封装 → 页面组件 → 配置路由
3. 统一返回格式、异常处理、权限控制
4. 代码创建时候需要使用skill `glue-micro-service` 创建

### 7.2 代码质量检查
- Go：检查未处理错误、格式、空指针
- Vue：检查 TS 类型、未使用变量、代码规范

### 7.3 调试与修复
- 优先查看日志
- 定位前后端报错位置
- 给出最小可运行修复代码

---

## 8. 最佳实践
- Go 错误优先返回，不使用 panic
- Vue 组件拆分合理，不超过 300 行, 组件功能要单一
- 数据库字段必须加注释
- 接口必须写注释（功能、入参、出参）
- 敏感接口必须加 JWT 验证

---

