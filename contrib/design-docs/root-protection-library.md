# Change Request

## **Short Summary**

Introduce a type which wraps a trusted root directory and offers methods that confine operations within that root.

## **Objective**

Many paths in the codebase should be interpreted within a given root, rather than in the whole filesystem. Currently, there is no unified approach to ensure this happens, and overlooking the risk may introduce path traversal vulnerabilities. A common case is the use of purely lexical joining with `filepath.Join`, as seen in [CVE-2026-55686](https://github.com/podman-container-tools/podman/security/advisories/GHSA-q6r4-3wmg-fwcq) and [CVE-2025-9566](https://github.com/podman-container-tools/podman/security/advisories/GHSA-wp3j-xq48-xpjw).

There are known solutions, such as [securejoin.SecureJoin](https://pkg.go.dev/github.com/cyphar/filepath-securejoin#SecureJoin), [pathrs-lite.OpenInRoot](https://pkg.go.dev/github.com/cyphar/filepath-securejoin/pathrs-lite#OpenInRoot), [os.Root](https://pkg.go.dev/os#Root). Additionally, such operations can be implemented manually with syscalls like `openat2`, for example [chunked in c/storage](https://github.com/podman-container-tools/container-libs/blob/fa0afc2957aace7e00cece1a2fe90544a467a8cb/storage/pkg/chunked/filesystem_linux.go#L351). However, it is left to the contributor to evaluate whether the risk is there, and then to reach for the right mechanism. This proposal aims to leverage the type system to sustainably shift this dynamic: path confinement becomesthe default, and bypassing it a deliberate, visible choice.

## **Detailed Description:**

<!--
How should the feature be implemented?
What considerations are there for the project if this is done?
-->

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
- [ ] Podman Developers
- [ ] Buildah Users
- [ ] Buildah Developers
- [ ] Skopeo Users
- [ ] Skopeo Developers
- [ ] Podman Desktop
- [ ] CRI-O
- [ ] Storage library
- [ ] Image library
- [ ] Common library
- [ ] Netavark and aardvark-dns

## ** Assignee(s) **

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
