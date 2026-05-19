# Advanced Querying & Filtering

conduit-cli's `describe` command allows for [jsonpath filtering](https://en.wikipedia.org/wiki/JSONPath). This can make conduit-cli a useful tool for scripting.

## Examples

```sh
# get the Transfer IDs of all transfers
client:$ conduit describe --jsonpath $..transferID
["3881b460-f415-4351-a69e-ba530d4cb341","202e1d65-8a76-4fb7-859d-58166560268c","9a73362d-223e-49aa-ac9b-3dc0e0028546"]

# get the Transfer ID of the most recent transfer
client:$ conduit describe --jsonpath x[0].transferID
"3881b460-f415-4351-a69e-ba530d4cb341"

# get the state of the Transfer with the TransferID of 3881b460-f415-4351-a69e-ba530d4cb341
client:$ conduit describe --jsonpath '$[? @.transferID=="3881b460-f415-4351-a69e-ba530d4cb341"].state'
["TRANSFER_VALIDATION_SUBMITTED"]

# get the transfer IDs of transfers in an error state
client:$ conduit describe --jsonpath '$[? @.state=="TRANSFER_ERROR"].transferID'
["3881b460-f415-4351-a69e-ba530d4cb341"]

```
