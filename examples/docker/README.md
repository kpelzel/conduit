## Helpful Commands

query etcd database directly
```bash
docker exec etcd-1 etcdctl --cert=/conduit/etcd_client_cert.pem --key=/conduit/etcd_client_key.pem --cacert=/conduit/conduit_internal_ca.pem --endpoints=https://192.168.20.21:2379 get --prefix transfers
```