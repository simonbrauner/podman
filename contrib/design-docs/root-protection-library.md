# Change Request

## **Short Summary**

Introduce a type which wraps a trusted root directory and offers methods that confine operations within that root.

## **Objective**

Many paths in the codebase should be interpreted within a given root, rather than in the whole filesystem. Currently, there is no unified approach to ensure this happens, and overlooking the risk may introduce path traversal vulnerabilities. A common case is the use of purely lexical joining with `filepath.Join`, as seen in [CVE-2026-55686](https://github.com/podman-container-tools/podman/security/advisories/GHSA-q6r4-3wmg-fwcq) and [CVE-2025-9566](https://github.com/podman-container-tools/podman/security/advisories/GHSA-wp3j-xq48-xpjw).

There are known solutions, such as [securejoin.SecureJoin](https://pkg.go.dev/github.com/cyphar/filepath-securejoin#SecureJoin), [pathrs-lite.OpenInRoot](https://pkg.go.dev/github.com/cyphar/filepath-securejoin/pathrs-lite#OpenInRoot), [os.Root](https://pkg.go.dev/os#Root). Additionally, such operations can be implemented manually with syscalls like `openat2`, for example [chunked in c/storage](https://github.com/podman-container-tools/container-libs/blob/fa0afc2957aace7e00cece1a2fe90544a467a8cb/storage/pkg/chunked/filesystem_linux.go#L351). However, it is left to the contributor to evaluate whether the risk is there, and then to reach for the right mechanism. This proposal aims to leverage the type system to sustainably shift this dynamic: path confinement becomesthe default, and bypassing it a deliberate, visible choice.

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

## **Use cases**

<!--
One or more short descriptions of use cases of the feature once complete.
-->

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
