生成 etcd-server 证书
```go
serverCert, serverKey, err := GenerateSignedCert(
    caCert,
    caKey,
    "etcd-server",
    []net.IP{
        net.ParseIP("192.168.10.10"),
        net.ParseIP("127.0.0.1"),
    },
    []x509.ExtKeyUsage{
        x509.ExtKeyUsageServerAuth,
    },
)
```

✔ 用于：
```shell
--cert-file
--key-file
```

4️⃣ 生成 etcd-peer 证书（⚠️ 必须同时支持 client + server）
```go
peerCert, peerKey, err := GenerateSignedCert(
    caCert,
    caKey,
    "etcd-peer",
    []net.IP{
        net.ParseIP("192.168.10.10"),
    },
    []x509.ExtKeyUsage{
        x509.ExtKeyUsageServerAuth,
        x509.ExtKeyUsageClientAuth,
    },
)
```

✔ 用于：
```shell
--peer-cert-file
--peer-key-file
```

5️⃣ 生成 etcd-client（给 kube-apiserver）
```go
clientCert, clientKey, err := GenerateSignedCert(
    caCert,
    caKey,
    "etcd-client",
    nil,
    []x509.ExtKeyUsage{
        x509.ExtKeyUsageClientAuth,
    },
)
```

✔ 用于 apiserver 参数：
```shell
--etcd-certfile
--etcd-keyfile
--etcd-cafile
```

四、etcd 启动参数如何对应（闭环）
```shell
etcd \
  --name node1 \
  --listen-client-urls=https://0.0.0.0:2379 \
  --advertise-client-urls=https://192.168.10.10:2379 \
  --listen-peer-urls=https://0.0.0.0:2380 \
  --initial-advertise-peer-urls=https://192.168.10.10:2380 \
  --cert-file=server.pem \
  --key-file=server-key.pem \
  --peer-cert-file=peer.pem \
  --peer-key-file=peer-key.pem \
  --trusted-ca-file=ca.pem \
  --peer-trusted-ca-file=ca.pem \
  --client-cert-auth \
  --peer-client-cert-auth
```

kube-apiserver（server cert）

必须满足：
```shell
ExtKeyUsage = serverAuth

SAN 包含：

所有 master IP

LB / VIP

kubernetes

kubernetes.default

kubernetes.default.svc

Service IP（如 10.96.0.1）

kubelet（client cert）

这是最容易错的

Subject:
  CN = system:node:<nodeName>
  O  = system:nodes


否则：

kubelet 能连 apiserver

但 永远注册不上节点

controller / scheduler（client cert）
CN = system:kube-controller-manager
CN = system:kube-scheduler

admin / kubectl
CN = admin
O  = system:masters
```