
A server, NAS navigation panel, Homepage, Browser homepage.
<br>
一个服务器、NAS导航面板、Homepage、浏览器首页。

</div>

## 🐳 Docker 部署（快速开始）

> 本仓库基于 `sun-panel` v1.8.1 构建，额外提供「完整数据备份 / 恢复」功能。
> 镜像由 GitHub Actions 自动构建，推送到 GitHub Container Registry（GHCR）。

**镜像地址**

```
ghcr.io/carolkenn/hc-panel:1.8.1-export
```

> [!NOTE]
> GHCR 的镜像包默认是私有的。首次拉取前请先登录，密码处填 GitHub Personal Access Token（需 `read:packages` 权限）：
>
> ```bash
> docker login ghcr.io -u <你的GitHub用户名>
> ```
>
> 如果想免登录拉取，可在仓库的 Package 设置里把可见性改为 `Public`。

### 方式一：docker compose（推荐）

新建 `docker-compose.yml`，内容如下：

```yaml
services:
  hc-panel:
    image: ghcr.io/carolkenn/hc-panel:1.8.1-export
    container_name: hc-panel
    volumes:
      - ./conf:/app/conf           # 配置文件（端口、数据库、上传路径）
      - ./uploads:/app/uploads     # 上传的图标 / 壁纸等文件
      - ./database:/app/database   # SQLite 数据库
    ports:
      - 3002:3002
    restart: always
```

启动：

```bash
docker compose up -d
```

### 方式二：docker run

```bash
docker run -d \
  --name hc-panel \
  --restart always \
  -p 3002:3002 \
  -v $(pwd)/conf:/app/conf \
  -v $(pwd)/uploads:/app/uploads \
  -v $(pwd)/database:/app/database \
  ghcr.io/carolkenn/hc-panel:1.8.1-export
```

> Windows PowerShell 中把 `$(pwd)` 换成绝对路径，例如 `-v D:/hc-panel/conf:/app/conf`。

### 访问与初始账号

浏览器打开 `http://<服务器IP>:3002`，初始管理员账号：

| 账号 | 密码 |
| ------------ | ------------ |
| `admin@sun.cc` | `12345678` |

首次登录后请立刻修改密码。如需改端口，编辑 `./conf/conf.ini` 中的 `[base] http_port` 并重启容器。

### 目录挂载说明

| 宿主机目录 | 容器内路径 | 说明 |
| ------------ | ------------ | ------------ |
| `./conf` | `/app/conf` | 配置文件 `conf.ini`，首次启动自动生成 |
| `./database` | `/app/database` | SQLite 数据库文件 |
| `./uploads` | `/app/uploads` | 上传的图标、壁纸等静态文件 |

> 这三个目录必须持久化。**数据导出功能生成的备份包内含 `files/` 与 `uploads/` 的相对路径一一对应，只备份数据库会丢失图片资源。**

### 升级与备份

```bash
docker compose pull
docker compose up -d
```

升级前建议先进入面板的「导入导出 → 完整数据备份」导出一份 ZIP，出问题时可原地恢复。

### 从源码构建（可选）

```bash
git clone https://github.com/CarolKenn/hc-panel.git
cd hc-panel
docker build -t hc-panel:1.8.1-export .
```





## 🐳 Deployment tutorial
本仓库的 Docker 部署方式见文首「🐳 Docker 部署（快速开始）」一节；
群晖、威联通、宝塔等其他平台的详细教程可参考官方文档 [Deployment Tutorial](https://sun-panel-doc.enianteam.com/usage/quick_deploy.html)。

