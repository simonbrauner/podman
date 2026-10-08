# Change Request

## **Short Summary**

Introduce a type which wraps a trusted root directory and offers methods that confine operations within that root.

## **Objective**

Many paths in the codebase should be interpreted within a given root, rather than in the whole filesystem. Currently, there is no unified approach to ensure this happens, and overlooking the risk may introduce path traversal vulnerabilities. A common case is the use of purely lexical joining with `filepath.Join`, as seen in [CVE-2026-55686](https://github.com/podman-container-tools/podman/security/advisories/GHSA-q6r4-3wmg-fwcq) and [CVE-2025-9566](https://github.com/podman-container-tools/podman/security/advisories/GHSA-wp3j-xq48-xpjw).

There are known solutions, such as [securejoin.SecureJoin](https://pkg.go.dev/github.com/cyphar/filepath-securejoin#SecureJoin), [pathrs-lite.OpenInRoot](https://pkg.go.dev/github.com/cyphar/filepath-securejoin/pathrs-lite#OpenInRoot), [os.Root](https://pkg.go.dev/os#Root). Additionally, operations can be implemented manually with syscalls like `openat2`, for example [chunked in c/storage](https://github.com/podman-container-tools/container-libs/blob/fa0afc2957aace7e00cece1a2fe90544a467a8cb/storage/pkg/chunked/filesystem_linux.go#L351). However, it is left to the contributor to evaluate whether the risk is there, and then to reach for the right mechanism. This proposal aims to leverage the type system to sustainably shift this dynamic: path confinement becomesthe default, and bypassing it a deliberate, visible choice.

## **Detailed Description:**

### **The basic idea**

#### String inside

```go
type PathRoot struct {
	path string
}
```

Roots in the codebase are almost exclusively represented as strings. Keeping the inner representation makes the migration **incremental** and **compiler-guided**:

- **Incremental** - It is not necessary to rework the logic of the migrated site beforehand. The root (e.g. in a struct field) can be retyped, and during the migration, the surrounding code that does not yet use the type keeps working through the underlying string. The switch to `PathRoot` can therefore happen at an arbitrary layer, which also gives us the flexibility to keep a stable API where desirable.

- **Compiler-guided** — Once the root is retyped, the previous string operations on it (such as `filepath.Join(root, ...)`) stop compiling. The compiler errors mark every place that needs attention, no usage is forgotten.

#### Constructor

```go
func NewPathRoot(path string) *PathRoot {
	return &PathRoot{path: path}
}
```

It is the responsibility of the caller to ensure that the root path is trusted.

**Open idea:** should the constructor itself perform some verification/preprocessing?

The constructor returns a pointer, as `nil` is a more idiomatic expression of the absence of a root than the empty string `""`. The existing `root == ""` checks get replaced with `root == nil`.

**Open idea:** the constructor should not allow `path` to be an empty string, is it sensible to panic on such an attempt?

#### Joining

```go
func (pr *PathRoot) Join(untrustedPath string) (string, error) {
	return securejoin.SecureJoin(pr.path, untrustedPath)
}
```

The general, cross-platform way to join an untrusted path onto the root, implemented by [`securejoin.SecureJoin`](https://pkg.go.dev/github.com/cyphar/filepath-securejoin#SecureJoin). It protects against non-TOCTOU (static) path traversal, but it is not TOCTOU-safe. Where a TOCTOU-safe variant exists it is generally preferred (primarily for security, possibly also for performance). `Join` is a reasonable middle ground for spots that have no TOCTOU-safe variant, or where migrating to one is deferred.

#### Getting the string

```go
func (pr *PathRoot) PathWithoutProtection() string {
	return pr.path
}
```

Returns the underlying string, for use cases which cannot be handled by the rest of the library as of now. It deliberately has a long and discouraging name, so that it makes a person think about the alternatives before calling it, and so that it is visible in code reviews. Because every call is greppable, the full set of such bypasses stays discoverable — a basis for auditing them and for possible checks on CI.

#### Logging

```go
func (pr *PathRoot) Format(f fmt.State, verb rune) {
	fmt.Fprintf(f, fmt.FormatString(f, verb), pr.path)
}
```

The `Stringer` interface is not implemented because it would inherently introduce a less visible alternative to `PathWithoutProtection`. To support format strings (e.g. for logging), `Formatter` is implemented instead. `Format` does not return the string, and `path := fmt.Sprintf("%s", root)` is less legitimate-looking than `path := root.String()`.

### **TOCTOU-safe extensions**

#### Limitations of string joining

The `Join` method defined above returns a resolved string. Between resolution and usage, the state of the file system may change, and if that happens, the joined path may no longer be confined. These are TOCTOU (time-of-check to time-of-use) vulnerabilities, and eliminating them requires support from the operating system to make resolution and the operation happen atomically.

#### No universal counterpart

Whereas static path traversal can be mitigated universally and cross-platform through the string-based `Join`, there is no single TOCTOU-safe implementation that covers the use cases in the code base:

- [os.Root](https://pkg.go.dev/os#Root) has the `RESOLVE_BENEATH` semantic, which rejects symlinks pointing outside the root, while for containers, the usual intent is `RESOLVE_IN_ROOT`, which re-roots such symlinks against the given root rather than the global `/`.
- [pathrs-lite.OpenInRoot](https://pkg.go.dev/github.com/cyphar/filepath-securejoin/pathrs-lite#OpenInRoot) and `openat2` provide the desired resolution type, but are Linux-specific, which implies a trade-off. Either use them only in Linux-specific code, or accept lower security guarantees on other platforms.

Operating on file descriptors also introduces the overhead of managing their life cycles, regardless of the underlying implementation. And a file descriptor is ephemeral, process-local state, whereas roots can be serialized and persisted.

With these drawbacks in mind, an fd-only approach is limited to the sites where a suitable fd-based implementation is available. It risks leaving the easier-to-exploit static traversal cases unprotected while the harder cases are addressed, and still relies on a string wherever the fd-based alternative is not feasible. Establishing the string baseline first gives static-traversal safety at a lower migration cost. From there, TOCTOU safety can be adopted incrementally, driven by prioritization and by which operations each platform supports.

#### Placement of TOCTOU-safe methods

##### a) On `PathRoot` directly

For cases where only one call is needed and it is therefore simpler to just call a function.

```go
func (pr *PathRoot) OpenFile(unsafePath string, mode os.FileMode) (*os.File, error) {
	...
}
```

##### b) On a separate type constructed from `PathRoot`

For cases where it is intended to use the operations multiple times, so that they can share one open.

```go
func (pr *PathRoot) OpenContainerRoot() (*ContainerRoot, error) {
	...
	return &ContainerRoot{file: file}, nil
}

type ContainerRoot struct {
	file *os.File
}

func (cr *ContainerRoot) OpenFile(unsafePath string, mode os.FileMode) (*os.File, error) {
	...
}

func (cr *ContainerRoot) Close() error {
	return cr.file.Close()
}
```

#### Duality of security guarantees

Once a TOCTOU-safe operation is supported on a platform, restricting its use to the code specific to that platform would needlessly limit the reach of the security improvement, while exposing it everywhere through a single method that silently falls back to `SecureJoin` would make it unclear whether TOCTOU safety holds for a given site.

The proposed solution is two explicitly named variants of each operation. The strict variant, suffixed `Concurrent`, guarantees TOCTOU safety and does not compile on unsupported platforms. The loose variant, suffixed `Racy`, is used where TOCTOU safety is not critical but still beneficial, and its implementation falls back to the string-based approach.

```go
// Only for linux
func (pr *PathRoot) OpenFileConcurrent(unsafePath string, mode os.FileMode) (*os.File, error) {
	...
}

// For all platforms
func (pr *PathRoot) OpenFileRacy(unsafePath string, mode os.FileMode) (*os.File, error) {
	if linux {
		return pr.OpenFileConcurrent(unsafePath, mode)
	} else {
		path, err := pr.Join(unsafePath)
		if err != nil {
			return nil, err
		}
		...
	}
}
```

#### Implementation used in methods

This is left open. A method can be backed by an external library (e.g. `pathrs-lite`) or self-implemented as in [chunked in c/storage](https://github.com/podman-container-tools/container-libs/blob/fa0afc2957aace7e00cece1a2fe90544a467a8cb/storage/pkg/chunked/filesystem_linux.go#L351).

### **Logistics**

#### Proposed first steps

- Migration of one root to make it static path traversal safe. In the proof of concept, this is [Mountpoint in libpod](https://github.com/podman-container-tools/podman/blob/1246f0ab8e8f627fbc27790ba648d5aea614b36c/libpod/container.go#L144), but it can be a different one.
- Moving the `Join` calls closer to the usage of their values, selecting a few TOCTOU-safe methods to support, and using them instead of the joined strings.
- Continuing in whichever direction provides the most value.

#### Placement of the library

Initially, the library with the type can reside in the codebase of the first migration, so that changes can be applied to it directly. Later on, it can move somewhere where it could be reused across the project, such as in `c/storage`.

## **Use cases**

The type(s) can be applied to struct fields, variables, and function signatures that represent roots which other paths should be confined against (e.g. [Mountpoint in libpod](https://github.com/podman-container-tools/podman/blob/1246f0ab8e8f627fbc27790ba648d5aea614b36c/libpod/container.go#L144)).

## **Target Podman Release**

The scope is large and this is incremental effort, so the adoption can span multiple releases, not one in particular.

In terms of hard deadlines, I personally am going to actively work on this until around mid-December 2026.

## **Link(s)**

<!--
A list of links to relevant context.
This can include Github issues describing the problem, related previous pull requests, or any other links that assist in understanding this change.
The use of non-Github issue trackers - e.g. corporate or distribution Jira or Bugzilla instances - is allowed, but we ask that all links here be publicly accessible to ensure full context is available to all.
Including a description with each link is not mandatory but is encouraged.
-->

## **Stakeholders**

<!--
A list of stakeholders who will be affected by this change.
Please check any boxes that apply.
For non-obvious stakeholders, you can add a brief sentence justifying after the checklist, but this is purely optional.
-->
- [ ] Podman Users
- [x] Podman Developers
- [ ] Buildah Users
- [x] Buildah Developers
- [ ] Skopeo Users
- [ ] Skopeo Developers
- [ ] Podman Desktop
- [ ] CRI-O
- [x] Storage library
- [ ] Image library
- [ ] Common library
- [ ] Netavark and aardvark-dns

## **Assignee(s)**

@simonbrauner

## **Impacts**

### **CLI**

<!--
Will there be any impact to the CLI?
Do any options need to be added?
Mocked output is strongly encouraged to help demonstrate the changes.
-->

### **Libpod**

<!--
Will there be any changes to the core container management logic?
-->

### **Others**

<!--
Are there any major impacts not mentioned above?
-->

## **Further Description (Optional):**

<!--
Is there anything not covered above that needs to be mentioned?
-->

## **Test Descriptions (Optional):**

<!--
How will this feature be tested?
Detail which existing test suite or suites will be used (or, if a new suite is required, why this is necessary and how it will be implemented.
Will this change require changes to the CI images (e.g. the inclusion of new packages) to be tested?
-->
