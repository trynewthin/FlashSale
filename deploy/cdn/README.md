# CDN 静态资源 & Media Store 文件管理

## 架构

```
┌──────────────────┐        ┌──────────────────────┐
│       cdn        │        │     media-store       │
│   (Nginx 只读)   │        │  (Go HTTP 读写服务)    │
│   端口: 19000    │        │   端口: 19001         │
└────────┬─────────┘        └──────────┬────────────┘
         │                             │
         └────────────┬────────────────┘
                      │
               共享 Volume
         deploy/cdn/assets/
         ├── products/     ← 商品主图
         └── banners/      ← 活动横幅（预留）
```

## CDN（只读，无鉴权）

对外提供静态文件访问。

```
GET http://localhost:19000/assets/products/xxx.svg
GET http://localhost:19000/assets/banners/yyy.jpg
```

## Media Store（读写，需鉴权）

提供文件增删查改 API，供管理员前端使用。

### 鉴权

所有 API 请求需携带 `Authorization: Bearer <secret>`。
默认 secret：`flashsale-media-dev`（通过 `FLASH_MEDIA_STORE_SECRET` 环境变量配置）。

### API

```
POST   /api/files/upload?category=products
       请求体: multipart/form-data, field name = "file"
       返回: { filename, category, url, size, created_at }

GET    /api/files?category=products
       返回: { files: [...], total: N }

GET    /api/files
       返回: 所有分类下的文件列表

DELETE /api/files/:category/:filename
       返回: { message: "deleted" }

GET    /api/categories
       返回: { categories: [{ name, count }] }

GET    /healthz
       返回: { status: "ok" }（无需鉴权）
```

### 示例

```bash
# 上传商品图片
curl -X POST "http://localhost:19001/api/files/upload?category=products" \
  -H "Authorization: Bearer flashsale-media-dev" \
  -F "file=@./product-photo.jpg"

# 列出商品图片
curl "http://localhost:19001/api/files?category=products" \
  -H "Authorization: Bearer flashsale-media-dev"

# 删除文件
curl -X DELETE "http://localhost:19001/api/files/products/1234_photo.jpg" \
  -H "Authorization: Bearer flashsale-media-dev"
```

## 部署说明

| 环境变量 | 用途 | 默认值 |
|---------|------|--------|
| `FLASH_CDN_PORT` | CDN 宿主机端口 | 19000 |
| `FLASH_MEDIA_STORE_PORT` | Media Store 宿主机端口 | 19001 |
| `FLASH_MEDIA_STORE_SECRET` | 鉴权 secret | flashsale-media-dev |
| `FLASHSALE_CDN_ORIGIN` | CDN 外部访问地址（用于生成 URL） | http://localhost:19000 |

服务器部署时，修改 `FLASHSALE_CDN_ORIGIN` 为服务器公网地址即可。
