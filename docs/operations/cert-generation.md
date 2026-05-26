# Generating Certificates

Certificates and keys can be generated using any utility capable of creating and signing x509 certificates. Conduit includes a built-in certificate generator to simplify this process.

## Security Architecture

Conduit uses two separate Certificate Authorities (CAs) to secure different parts of the system:

**External CA** - Secures client-to-server communication:

- Client requests to the Conduit server (conduit-cli, API clients)
- User authentication and authorization

**Internal CA** - Secures internal component communication:

- ETCD cluster
- rqlite cluster
- conduit-runner
- conduit-fta

## Examples

### Conduit Internal CA

```sh
./conduit-server internal-ca -d \
    --internal-ca-cert ./conduit_internal_ca.pem \
    --internal-ca-key ./conduit_internal_key.pem \
```

### Conduit External CA

```sh
./conduit-server external-ca -d \
    --external-ca-cert ./conduit_external_ca.pem \
    --external-ca-key ./conduit_external_key.pem \
```

### ETCD & Rqlite Server Cert & Key

Admins need to provide these generated cert & keys to etcd and rqlite when setting up conduit.

```sh
./conduit-server internal-server-cert -d \
    --internal-ca-cert ./conduit_internal_ca.pem \
    --internal-ca-key ./conduit_internal_key.pem \
    --separate-cert-key \
    --cert-name etcd_server_cert.pem \
    --key-name etcd_server_key.pem \
    --output ./ \
    --server-ip 192.168.20.21,192.168.20.22,192.168.20.23 \
    --server-hostname etcd-1.example.com,etcd-2.example.com,etcd-3.example.com

./conduit-server internal-server-cert -d \
    --internal-ca-cert ./conduit_internal_ca.pem \
    --internal-ca-key ./conduit_internal_key.pem \
    --separate-cert-key \
    --cert-name rqlite_server_cert.pem \
    --key-name rqlite_server_key.pem \
    --output ./ \
    --server-ip 192.168.20.31,192.168.20.32,192.168.20.33 \
    --server-hostname rqlite-1.example.com,rqlite-2.example.com,rqlite-3.example.com
```

### Conduit Admin & Slurm Plugin Cert & Key

This admin key can be used with `conduit` to get and control transfers as if they were the user.

The slurm cert and key need to be provided to the slurm lua plugin so it can authenticate with conduit.

```sh
./conduit-server external-client-cert -d \
    --external-ca-cert ./conduit_external_ca.pem \
    --external-ca-key ./conduit_external_key.pem \
    --separate-cert-key \
    --cert-name conduit_admin_cert.pem \
    --key-name conduit_admin_key.pem \
    --output ./ \
    --client-commonname conduit-admin \
    --expiration 365

./conduit-server external-client-cert -d \
    --external-ca-cert ./conduit_external_ca.pem \
    --external-ca-key ./conduit_external_key.pem \
    --separate-cert-key \
    --cert-name conduit_slurm_cert.pem \
    --key-name conduit_slurm_key.pem \
    --output ./ \
    --client-commonname conduit-service \
    --expiration 365
```

### Conduit Etcd & Rqlite Client Cert & Key

These are optional certs that can be used to talk directly to etcd or rqlite using their respective clients

```sh
./conduit-server internal-client-cert -d \
    --internal-ca-cert ./conduit_internal_ca.pem \
    --internal-ca-key ./conduit_internal_key.pem \
    --separate-cert-key \
    --cert-name etcd_client_cert.pem \
    --key-name etcd_client_key.pem \
    --output ./ \
    --client-commonname root \
    --expiration 365

./conduit-server internal-client-cert -d \
    --internal-ca-cert ./conduit_internal_ca.pem \
    --internal-ca-key ./conduit_internal_key.pem \
    --separate-cert-key \
    --cert-name rqlite_client_cert.pem \
    --key-name rqlite_client_key.pem \
    --output ./ \
    --client-commonname "" \
    --expiration 365
```
