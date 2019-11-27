% CONTAINERS-REGISTRIES.CONF(5) System-wide registry configuration file
% Brent Baude
% Aug 2017

# NAME
containers-registries.conf - Syntax of System Registry Configuration File

# DESCRIPTION
The CONTAINERS-REGISTRIES configuration file is a system-wide configuration
file for container image registries. The file format is TOML.

By default, the configuration file is located at `/etc/containers/registries.conf`.

# FORMATS

## VERSION 2
VERSION 2 is the latest format of the `registries.conf` and is currently in
beta. This means in general VERSION 1 should be used in production environments
for now.

### GLOBAL SETTINGS

`unqualified-search-registries`
: An array of _host_[`:`_port_] registries to try when pulling an unqualified image, in order.

### NAMESPACED `[[registry]]` SETTINGS

The bulk of the configuration is represented as an array of `[[registry]]`
TOML tables; the settings may therefore differ among different registries
as well as among different namespaces/repositories within a registry.

#### Choosing a `[[registry]]` TOML table

Given an image name, a single `[[registry]]` TOML table is chosen based on its `prefix` field.

`prefix`
: A prefix of the user-specified image name, i.e. using one of the following formats:
    - _host_[`:`_port_]
    - _host_[`:`_port_]`/`_namespace_[`/`_namespace_…]
    - _host_[`:`_port_]`/`_namespace_[`/`_namespace_…]`/`_repo_
    - _host_[`:`_port_]`/`_namespace_[`/`_namespace_…]`/`_repo_(`:`_tag|`@`_digest_)

    The user-specified image name must start with the specified `prefix` (and continue
    with the appropriate separator) for a particular `[[registry]]` TOML table to be
    considered; (only) the TOML table with the longest match is used.

    As a special case, the `prefix` field can be missing; if so, it defaults to the value
    of the `location` field (described below).

#### Per-namespace settings

`insecure`
: `true` or `false`.
    By default, container runtimes require TLS when retrieving images from a registry.
    If `insecure` is set to `true`, unencrypted HTTP as well as TLS connections with untrusted
    certificates are allowed.

`blocked`
: `true` or `false`.
    If `true`, pulling images with matching names is forbidden.

#### Remapping and mirroring registries

The user-specified image reference is, primarily, a "logical" image name, always used for naming
the image.  By default, the image reference also directly specifies the registry and repository
to use, but the following options can be used to redirect the underlying accesses
to different registry servers or locations (e.g. to support configurations with no access to the
internet without having to change `Dockerfile`s, or to add redundancy).

`location`
: Accepts the same format as the `prefix` field, and specifies the physical location
    of the `prefix`-rooted namespace.

    By default, this equal to `prefix` (in which case `prefix` can be omitted and the
    `[[registry]]` TOML table can only specify `location`).

    Example: Given
    ```
    prefix = "example.com/foo"
    location = "internal-registry-for-example.net/bar"
    ```
    requests for the image `example.com/foo/myimage:latest` will actually work with the
    `internal-registry-for-example.net/bar/myimage:latest` image.

`mirror`
: An array of TOML tables specifying (possibly-partial) mirrors for the
    `prefix`-rooted namespace.

    The mirrors are attempted in the specified order; the first one that can be
    contacted and contains the image will be used (and if none of the mirrors contains the image,
    the primary location specified by the `registry.location` field, or using the unmodified
    user-specified reference, is tried last).

    Each TOML table in the `mirror` array can contain the following fields, with the same semantics
    as if specified in the `[[registry]]` TOML table directly:
    - `location`
    - `insecure`

`mirror-by-digest-only`
: `true` or `false`.
    If `true`, mirrors will only be used during pulling if the image reference includes a digest.
    Referencing an image by digest ensures that the same is always used
    (whereas referencing an image by a tag may cause different registries to return
    different images if the tag mapping is out of sync).

    Note that if this is `true`, images referenced by a tag will only use the primary
    registry, failing if that registry is not accessible.

*Note*: Redirection and mirrors are currently processed only when reading images, not when pushing
to a registry; that may change in the future.

### EXAMPLE

```
unqualified-search-registries = ["example.com"]

[[registry]]
prefix = "example.com/foo"
insecure = false
blocked = false
location = "internal-registry-for-example.com/bar"

[[registry.mirror]]
location = "example-mirror-0.local/mirror-for-foo"

[[registry.mirror]]
location = "example-mirror-1.local/mirrors/foo"
insecure = true
```
Given the above, a pull of `example.com/foo/image:latest` will try:
    1. `example-mirror-0.local/mirror-for-foo/image:latest`
    2. `example-mirror-1.local/mirrors/foo/image:latest`
    3. `internal-registry-for-example.net/bar/myimage:latest`

in order, and use the first one that exists.

## VERSION 1
VERSION 1 can be used as alternative to the VERSION 2, but it does not support
using registry mirrors, longest-prefix matches, or location rewriting.

The TOML format is used to build a simple list of registries under three
categories: `registries.search`, `registries.insecure`, and `registries.block`.
You can list multiple registries using a comma separated list.

Search registries are used when the caller of a container runtime does not fully specify the
container image that they want to execute.  These registries are prepended onto the front
of the specified container image until the named image is found at a registry.

Note that insecure registries can be used for any registry, not just the registries listed
under search.

The `registries.insecure` and `registries.block` lists have the same meaning as the
`insecure` and `blocked` fields in VERSION 2.

### EXAMPLE
The following example configuration defines two searchable registries, one
insecure registry, and two blocked registries.

```
[registries.search]
registries = ['registry1.com', 'registry2.com']

[registries.insecure]
registries = ['registry3.com']

[registries.block]
registries = ['registry.untrusted.com', 'registry.unsafe.com']
```

# HISTORY
Mar 2019, Added additional configuration format by Sascha Grunert <sgrunert@suse.com>

Aug 2018, Renamed to containers-registries.conf(5) by Valentin Rothberg <vrothberg@suse.com>

Jun 2018, Updated by Tom Sweeney <tsweeney@redhat.com>

Aug 2017, Originally compiled by Brent Baude <bbaude@redhat.com>
