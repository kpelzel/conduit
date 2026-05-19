# LDAP Integration

Conduit supports user lookup with LDAP to provide flexible user identity translation. LDAP lookup translates between different user identifiers (Kerberos principals, usernames, and UID numbers) to determine the actual username (`uid` attribute) for file ownership and transfer operations.

## Overview

LDAP integration in Conduit serves two primary purposes:

1. **Kerberos to Unix Username Translation**: When users authenticate via Kerberos, LDAP maps the Kerberos principal (e.g., `user@REALM.COM`) to a local Unix username
2. **UID Number to Username Translation**: When privileged services (ex: conduit slurm plugin) provide a numeric UID instead of a username, LDAP resolves it to the corresponding username

## Configuration

LDAP configuration is defined in the main Conduit server configuration file under the `ldap` section:

```yaml
ldap:
  # The hostname or IP address of the LDAP server
  host: "ldap.example.com"
  
  # The port that Conduit will use to connect to the LDAP server
  # Standard LDAP port is 389, LDAPS is 636
  port: 389
  
  # A list of LDAP base DN(s) that Conduit will search to find a user
  # Conduit will search all specified base DNs
  base-dn:
    - "dc=example,dc=com"
    - "ou=users,dc=example,dc=com"
  
  # LDAP attributes to match against Kerberos principals (username@realm)
  # Used when authenticating via Kerberos
  krb5-attributes:
    - "krbPrincipalName"
    - "hpcKerbPrincipals"
  
  # LDAP attributes to match against the username portion of Kerberos principal
  # Used as a fallback or alternative matching method
  uname-attributes:
    - "uid"
    - "sAMAccountName"
  
  # LDAP attributes to match against numeric UID values
  # Used when privileged services provide a UID number instead of username
  uid-number-attributes:
    - "uidNumber"
```

## How It Works

### Search Process

When Conduit needs to resolve a user identity:

1. **Connection**: Conduit establishes a TCP connection to the LDAP server
2. **Search Execution**: For each configured base DN, Conduit executes an LDAP search with the appropriate filter
3. **Result Validation**: 
   - If all base DN searches fail, an error is returned
   - If no entries are found, an error is returned
   - If multiple entries are found with conflicting `uid` values, an error is returned
4. **Return**: The `uid` attribute value from the matched entry is returned as the username

#### Kerberos Principal Lookup
When a user authenticates with Kerberos (e.g., `jdoe@EXAMPLE.COM`):
```ldap
(|(krbPrincipalName=jdoe@EXAMPLE.COM)(hpcKerbPrincipals=jdoe@EXAMPLE.COM)(uid=jdoe)(sAMAccountName=jdoe))
```
This searches using both the full principal and just the username portion. These attributes are defined in the conduit config under `krb5-attributes` and `uname-attributes`.

#### UID Number Lookup
When a privileged service provides a UID number (e.g., `1001`):
```ldap
(|(uidNumber=1001))
```
These attributes are defined in the conduit config under `uid-number-attributes`.

### Usage Scenarios

#### Scenario 1: User Authenticates with Kerberos
```
1. User authenticates via Kerberos as: jdoe@EXAMPLE.COM
2. Conduit queries LDAP with krb5-attributes and uname-attributes
3. LDAP returns: uid=jdoe
4. All transfers run as user: jdoe
```

#### Scenario 2: Privileged Service Provides UID
```
1. Privileged service (e.g., conduit-runner) acts on behalf of UID: 1001
2. Conduit queries LDAP with uid-number-attributes
3. LDAP returns: uid=jdoe
4. Transfer runs as user: jdoe
```

#### Scenario 3: LDAP Not Configured
```
1. User authenticates via Kerberos as: jdoe@EXAMPLE.COM
2. LDAP is not configured
3. Conduit uses the Kerberos username directly: jdoe
4. Transfers run as user: jdoe
```

For UID lookups without LDAP, Conduit falls back to local system user lookup using the operating system's user database.

> **Note on Local User Lookup**: The behavior of local system user lookup depends on the CGO build configuration:
> - **CGO Enabled** (default): Uses the system's native user database (e.g., NSS, sssd, Active Directory integration, etc.). This provides full integration with enterprise authentication systems.
> - **CGO Disabled**: Only searches the `/etc/passwd` file directly. This has limited functionality and will not work with networked authentication systems.
>
> This applies to the Conduit server (when LDAP is not configured), conduit-runner, and conduit-fta . For production environments with enterprise authentication, ensure CGO is enabled during the build process of the conduit-runner and conduit-fta.

## Requirements

To enable LDAP integration:

1. **Minimum Required Configuration**:
   - `host`: Must be specified
   - `port`: Must be specified (typically 389 or 636)
   - `base-dn`: At least one base DN must be provided
   - At least one attribute list must be specified (krb5-attributes, uname-attributes, or uid-number-attributes)

2. **Network Access**: The Conduit server must have network connectivity to the LDAP server

3. **LDAP Schema**: The LDAP directory must contain the `uid` attribute for user entries (this is the standard Unix username attribute)

## Timeout and Performance

- **Default Timeout**: 30 seconds for LDAP operations
- **Search Scope**: Searches use `WholeSubtree` scope, recursively searching under each base DN
- **Connection Model**: A new LDAP connection is established for each lookup operation
- **Deref Aliases**: Alias dereferencing is disabled (`NeverDerefAliases`)

## Example Configuration

Here's a complete example for a typical LDAP setup:

```yaml
ldap:
  host: "ldap.example.com"
  port: 389
  base-dn:
    - "dc=example,dc=com"
  krb5-attributes:
    - "krbPrincipalName"
    - "hpcKerbPrincipals"
  uname-attributes:
    - "uid"
  uid-number-attributes:
    - "uidNumber"
```
