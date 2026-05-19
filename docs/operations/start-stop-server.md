# API Control

A conduit instance can be fully controlled using conduit-cli. This requires the use of an admin cert/key provided to the cli. A specific conduit instance can be targeted by providing flags for the ip and port of the instance you want to control.

### `conduit control drain`

This will prevent an instance of conduit from progressing a transfer from the 'init' state, but will continue all other transfers with later states. This is only intended to be used if _all_ conduit instances for a particular system are in the drain state, otherwise the draining conduit instance will continue to progress new transfers that other instances have moved from the 'init' state. Due to conduit's architecture, a single conduit instance does not know when all transfers have been drained, therefore conduit will not stop draining until an admin manually stops conduit with systemctl.

### `conduit control start`

This will resume conduit from a 'draining' state.

# systemd Control

Conduit fully supports starting and stopping with systemd.

### `systemctl stop conduit`

This will tell a conduit instance to shutdown. Any waiting leases will be reverted back to "validation complete" so another conduit instance can take it. Any current running transfers will continue to run so long as the conduit runner is not killed. Any new transfer requests will be rejected while the server is stopping.

### `systemctl start conduit`

This will start conduit from a fully stopped state. This is not the same as `conduit control start` as that is used for resuming from a `conduit control drain`
