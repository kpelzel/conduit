# **conduit-cli Usage**

```
conduit <COMMAND> [ARGS] [FLAGS]
```

View help for any command:

```
conduit --help
conduit <command> --help
```

---

# **Global Flags**

---

| Name                 | Type | Required | Description                                                                                        |
| -------------------- | ---- | -------- | -------------------------------------------------------------------------------------------------- |
| `--ca, -c`           | flag | no       | Location of conduit root CA                                                                        |
| `--cert`             | flag | no       | The path to the mTLS client cert to use for the request                                            |
| `--cert-key-bundle`  | flag | no       | shorthand for setting --cert and --key to the same path (default "~/.conduit-cert-key-bundle.pem") |
| `--config`           | flag | no       | config file (default is /home/<user>/conduit-cli-config.yaml)                                      |
| `--debug, -d`        | flag | no       | enable debugging                                                                                   |
| `--grpc-limit`       | flag | no       | The size limit (in bytes) of grpc messages received from conduit (default 100000000)               |
| `--ip, -i`           | flag | no       | Addr of the conduit server                                                                         |
| `--key`              | flag | no       | The path to the mTLS client key to use for the request                                             |
| `--krb-cache`        | flag | no       | Location of the krb5 ticket cache (default "/tmp")                                                 |
| `--krb-cache-prefix` | flag | no       | The Prefix before the UID of the tickets located in the krb5 cache (default "krb5cc\_")            |
| `--krb-config`       | flag | no       | Location of the krb5 config file (default "/etc/krb5.conf")                                        |
| `--krb-spn`          | flag | no       | The conduit spn (default "conduit/conduit-server.example.com")                                     |
| `--port, -p`         | flag | no       | Port of the conduit server (default 23456)                                                         |
| `--req-timeout`      | flag | no       | Timeout in seconds for requests made to conduit (default 1m)                                       |

---

# **Commands**

---

## **cp**

copy files and/or directories

### **Usage**

```
conduit cp SOURCE... DESTINATION [flags]
```

### **Arguments & Flags**

| Name                    | Type       | Required | Description                                                                    |
| ----------------------- | ---------- | -------- | ------------------------------------------------------------------------------ |
| `SOURCE`                | positional | yes      | path to source file(s)/directories                                             |
| `DESTINATION`           | positional | yes      | path to destination file/directory                                             |
| `--background, -b`      | flag       | no       | Submit a conduit transfer without watching it progress to completion           |
| `--quiet, -q`           | flag       | no       | Reduce command output to only a transferID                                     |
| `--recursive, -r`       | flag       | no       | Copy directories recursively                                                   |
| `--skip-validation, -s` | flag       | no       | Skip waiting for validation to succeed                                         |
| `--user`                | flag       | no       | The user to start the transfer as. Requires an admin cert & key to be provided |
| `--validate-only`       | flag       | no       | Do not transfer any data, just run validation                                  |

### **Example**

```
conduit cp /mnt/fs_1/file_1 /mnt/fs_1/file2 /mnt/fs_2/dest_dir
```

## **mv**

move files and/or directories

### **Usage**

```
conduit mv SOURCE... DESTINATION [flags]
```

### **Arguments & Flags**

| Name                    | Type       | Required | Description                                                                    |
| ----------------------- | ---------- | -------- | ------------------------------------------------------------------------------ |
| `SOURCE`                | positional | yes      | path to source file(s)/directories                                             |
| `DESTINATION`           | positional | yes      | path to destination file/directory                                             |
| `--background, -b`      | flag       | no       | Submit a conduit transfer without watching it progress to completion           |
| `--quiet, -q`           | flag       | no       | Reduce command output to only a transferID                                     |
| `--recursive, -r`       | flag       | no       | Move directories recursively                                                   |
| `--skip-validation, -s` | flag       | no       | Skip waiting for validation to succeed                                         |
| `--user`                | flag       | no       | The user to start the transfer as. Requires an admin cert & key to be provided |
| `--validate-only`       | flag       | no       | Do not transfer any data, just run validation                                  |
