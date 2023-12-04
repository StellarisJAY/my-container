# MyContainer
模仿Docker实现的容器工具

## 运行
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