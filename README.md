# rocket
**rocket** 是 `hextech` 平台的核心服务。负责所有 `CRD` 的管理工作

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
export ROCKET01=$(docker inspect rocket01-control-plane --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
export ROCKET02=$(docker inspect rocket02-control-plane --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
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

