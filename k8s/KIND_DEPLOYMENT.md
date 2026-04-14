# NetShop 在 kind 上的完整部署流程

本文档整理了本次实际可用的 Kubernetes 部署步骤，适用于本项目当前状态：

- 业务服务运行在 kind 集群
- PostgreSQL / Redis 仍由宿主机 docker-compose 提供
- frontend 通过 port-forward 先访问，Ingress 可后装

## 1. 前置条件

需要安装以下工具：

- Docker Desktop
- kubectl
- kind

确认工具可用：

```bash
docker --version
kubectl version --client
kind --version
```

## 2. 准备本地 Secret 文件

本项目使用本地文件生成 dev 环境 Secret，避免把敏感信息提交到仓库。

复制示例：

```bash
cp k8s/overlays/dev/secret-dev.local.env.example k8s/overlays/dev/secret-dev.local.env
```

编辑文件并填入真实测试值：

- JWT_SECRET
- GITHUB_CLIENT_ID
- GITHUB_CLIENT_SECRET
- ARK_API_KEY

注意：`GITHUB_CLIENT_ID` 不要有前后空格，否则 GitHub OAuth 链接会异常（client_id 前后出现 `+`）。

## 3. 创建或切换 kind 集群

创建集群（已存在可跳过）：

```bash
kind create cluster --name netshop
```

切换 kubectl 上下文：

```bash
kubectl config use-context kind-netshop
```

## 4. 构建 8 个服务镜像

在仓库根目录执行：

```bash
docker build -f services/user/Dockerfile -t netshop/user:latest .
docker build -f services/email/Dockerfile -t netshop/email:latest .
docker build -f services/product/Dockerfile -t netshop/product:latest .
docker build -f services/recommend/Dockerfile -t netshop/recommend:latest .
docker build -f services/ad/Dockerfile -t netshop/ad:latest .
docker build -f services/cart/Dockerfile -t netshop/cart:latest .
docker build -f services/aiassistant/Dockerfile -t netshop/aiassistant:latest .
docker build -f services/frontend/Dockerfile -t netshop/frontend:latest .
```

说明：

- Dockerfile 已改为使用可访问镜像源（m.daocloud.io）
- 必须以仓库根目录为 build context（命令最后是 `.`）

## 5. 导入镜像到 kind

```bash
kind load docker-image netshop/user:latest --name netshop
kind load docker-image netshop/email:latest --name netshop
kind load docker-image netshop/product:latest --name netshop
kind load docker-image netshop/recommend:latest --name netshop
kind load docker-image netshop/ad:latest --name netshop
kind load docker-image netshop/cart:latest --name netshop
kind load docker-image netshop/aiassistant:latest --name netshop
kind load docker-image netshop/frontend:latest --name netshop
```

## 6. 应用 K8s 资源

```bash
kubectl apply -k k8s/overlays/dev
```

检查资源：

```bash
kubectl get all -n netshop
```

等待 Deployment 就绪：

```bash
kubectl rollout status deployment/ad -n netshop
kubectl rollout status deployment/aiassistant -n netshop
kubectl rollout status deployment/cart -n netshop
kubectl rollout status deployment/email -n netshop
kubectl rollout status deployment/frontend -n netshop
kubectl rollout status deployment/product -n netshop
kubectl rollout status deployment/recommend -n netshop
kubectl rollout status deployment/user -n netshop
```

## 7. 访问方式（推荐先用 port-forward）

如果暂时不装 ingress-nginx，直接本地转发 frontend：

```bash
kubectl port-forward -n netshop svc/frontend 8080:80
```

浏览器访问：

```text
http://127.0.0.1:8080
```

## 8. 可选：安装 Ingress（会比较慢）

kind 环境安装 ingress-nginx：

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=300s
```

安装后可使用 Ingress 访问（按项目中的 host 配置）。

## 9. 常见问题排查

### 9.1 docker build 报 401 Unauthorized

现象示例：拉取 golang 基础镜像时报鉴权失败。

原因：本机 Docker 代理镜像源不可用或无权限。

处理：本项目 Dockerfile 已切换到 m.daocloud.io 镜像源，使用当前仓库代码重新构建即可。

### 9.2 GitHub 登录跳转 404

现象：访问 authorize 链接后 404。

常见原因：

- `GITHUB_CLIENT_ID` 有前后空格
- GitHub OAuth App 的 callback URL 与当前访问地址不一致

处理：

- 检查 `k8s/overlays/dev/secret-dev.local.env`
- 重新执行 `kubectl apply -k k8s/overlays/dev`
- 重启 frontend：`kubectl rollout restart deployment/frontend -n netshop`

## 10. 清理

删除业务资源：

```bash
kubectl delete -k k8s/overlays/dev
```

删除 kind 集群（可选）：

```bash
kind delete cluster --name netshop
```
