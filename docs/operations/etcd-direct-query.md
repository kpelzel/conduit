## How to access etcd records manually

to query etcd directly, use the etcdctl command. A cert/key thats been signed by the conduit internal CA is required to alter etcd key & values (see the Cert Generation doc for more details). Because etcd is a simple key-value store, conduit uses "prefixes" to organize the transfers. All transfers are under `transfers/` and further separated by their transfer-id `transfers/11111111-1111-1111-1111-111111111111`. From there it breaks down into the various fields in a transfer object (state, error, errormessage, etc). Here are some example commands:

```
# Get the value of that is at transfers/ebb444e4-48d8-4fc1-bc31-c8c6816f2b0d/state
etcdctl get --cert /etc/conduit/keys/etcd_client_cert.pem --key /etc/conduit/keys/etcd_client_key.pem --cacert /etc/conduit/keys/conduit_ca.pem --endpoints=192.168.0.254:2379 transfers/ebb444e4-48d8-4fc1-bc31-c8c6816f2b0d/state
```

```
# Get every key-value that is under a single transfer transfers/ebb444e4-48d8-4fc1-bc31-c8c6816f2b0d
etcdctl get --cert /etc/conduit/keys/etcd_client_cert.pem --key /etc/conduit/keys/etcd_client_key.pem --cacert /etc/conduit/keys/conduit_ca.pem --endpoints=192.168.0.254:2379 --prefix transfers/ebb444e4-48d8-4fc1-bc31-c8c6816f2b0d

```
