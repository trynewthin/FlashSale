# React + TypeScript + Vite + shadcn/ui

This is a template for a new Vite project with React, TypeScript, and shadcn/ui.

## Vercel 部署说明

1. 在 Vercel 中将项目 Root Directory 指向 `frontend/user`。
2. 配置环境变量 `BACKEND_ORIGIN`（示例见 `.env.production.example`）。
3. 保持 `VITE_USER_API_BASE_URL` 默认值 `/api/v1`（由 Vercel `api/[...path].js` 代理转发）。
4. `vercel.json` 已配置：
   - `/api/*` 走 Serverless 代理；
   - 其余路由回退到 `index.html`（支持 React Router 刷新）。
