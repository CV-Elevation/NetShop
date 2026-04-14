# NetShop Kubernetes 部署

完整实操流程请优先参考：[KIND_DEPLOYMENT.md](KIND_DEPLOYMENT.md)

本目录已包含以下资源：

- `base/`：通用资源
  - Namespace / ConfigMap / Secret
  - user, email, product, recommend, ad, cart, aiassistant, frontend 的 Deployment + Service
- `overlays/dev/`：开发环境覆盖
  - 本机 docker-compose 基础设施地址覆盖
  - frontend Ingress
  - 本地 kind 部署说明

## 1. 前置条件

- 已安装 `kubectl`
- 集群已可用
- 已安装 `ingress-nginx`（如需 Ingress 访问）
- 已安装 `kind`

## 2. 修改 dev 覆盖

1. 复制 Secret 示例文件为本地私有文件（不会提交）

```bash
cp k8s/overlays/dev/secret-dev.local.env.example k8s/overlays/dev/secret-dev.local.env
```

2. 编辑 `k8s/overlays/dev/secret-dev.local.env`，填入你的本机测试值

3. `k8s/overlays/dev/secret-dev.local.env` 已加入 `.gitignore`，避免误提交

## 2. 本地 kind 镜像流程

先在本地构建镜像，再导入 kind 集群：

```bash
docker build -f services/user/Dockerfile -t netshop/user:latest .
docker build -f services/email/Dockerfile -t netshop/email:latest .
docker build -f services/product/Dockerfile -t netshop/product:latest .
docker build -f services/recommend/Dockerfile -t netshop/recommend:latest .
docker build -f services/ad/Dockerfile -t netshop/ad:latest .
docker build -f services/cart/Dockerfile -t netshop/cart:latest .
docker build -f services/aiassistant/Dockerfile -t netshop/aiassistant:latest .
docker build -f services/frontend/Dockerfile -t netshop/frontend:latest .

kind load docker-image netshop/user:latest
kind load docker-image netshop/email:latest
kind load docker-image netshop/product:latest
kind load docker-image netshop/recommend:latest
kind load docker-image netshop/ad:latest
kind load docker-image netshop/cart:latest
kind load docker-image netshop/aiassistant:latest
kind load docker-image netshop/frontend:latest
```

## 3. 部署

```bash
kubectl apply -k k8s/overlays/dev
```

## 4. 验证

```bash
kubectl get all -n netshop
kubectl get ingress -n netshop
kubectl rollout status deployment/frontend -n netshop
```

## 5. 本机访问

若使用 `netshop.local`，请添加 hosts：

```bash
sudo sh -c 'echo "127.0.0.1 netshop.local" >> /etc/hosts'
```

然后访问：

```text
http://netshop.local
```

## 6. 清理

```bash
kubectl delete -k k8s/overlays/dev
```
