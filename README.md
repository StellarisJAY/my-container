# MyContainer
模仿Docker实现的容器工具

## 运行示例
拉取运行Redis，并使用宿主机的配置文件和限制CPU使用
```shell
# 编译go项目
make build 
# 拉取镜像
sudo ./my-container pull redis:latest
# 运行镜像，可设置CPU配额，可挂载目录到容器
sudo ./my-container run \
  -image redis:latest \
  -cpu 0.5 \
  -mount src=/etc/redis/redis.conf,dest=/etc/redis.conf \
  /usr/local/bin/redis-server \
  /etc/redis.conf 
```

## 命令示例
```shell
# list镜像
./my-container images
# 列出正在运行的容器
./my-container ps
# 在运行的容器中执行命令
./my-container exec -container {containerId} /bin/sh
```