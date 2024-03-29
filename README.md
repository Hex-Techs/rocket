# Rocket
**Rocket** 是 `hextech` 平台的核心服务。负责所有 `CRD` 的管理工作

## 快速开始
你需要一个Kubernetes集群来运行。 你可以使用 [k3d](https://k3d.io) 来运行一个本地集群进行测试。

### 通过以下命令生成 token
```sh
# linux
echo "$(head -c 6 /dev/urandom | md5sum | head -c 6)"."$(head -c 16 /dev/urandom | md5sum | head -c 16)"

# mac os
echo "$(head -c 6 /dev/urandom | md5 | head -c 6)"."$(head -c 16 /dev/urandom | md5 | head -c 16)"
```

### 获取集群的 APISERVER 地址
```sh
export ROCKET01=$(docker inspect k3d-rocket01-server-0 --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
export ROCKET02=$(docker inspect k3d-rocket02-server-0 --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
```

### 在集群里运行
1. . 通过指定 `IMG` 变量来构建并推送镜像:

```sh
make all IMG=<some-registry>/rocket:tag
```

2. 安装 `CRD` 到集群:

```sh
make install
```

3. 卸载 `CRD`

```sh
make uninstall
```

4. 安装 `OpenKruise`

针对 `k3s` 集群中安装 `OpenKruise` 需要指定配置 `--set daemon.socketLocation="/run/k3s"`

### Roadmap

- [x] 支持 Deployment
- [x] 支持 CloneSet
- [x] 支持 CronJob
- [ ] 支持 StatefulSet
- [ ] 支持 kruise StatefulSet
- [ ] 支持 Job 


## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

