# versionctl

<p align="center">
    <em>Version management without the fuss</em>
</p>

---

_versionctl_ (short for version control) is a binary utility that uses semantic commits, your own git history and some light configuration to manage your application's version.

It **does**:

- Calculate your next application version
- Convert version between formats (e.g., git tag, docker tag, node version)
- Writes version to files (with special handling for known project files)

It **does not**:

- Integrate with remote VCS
- Create commits or tags on your behalf
- Generate changelogs from commit history

This is because all-in-one semantic-release solutions already exist - this tool helps you manage the version of your application while still letting you control your release process.

## Installation

_versionctl_ is a single binary you can download from the [releases](https://github.com/benfiola/versionctl/releases) page.

Currently, binaries are produced for the following platforms:

- linux-arm64
- linux-amd64
- darwin-arm64
- darwin-amd64

Installation is as simple as downloading the binary, making it executable and using it:

```shell
$ os="linux" arch="arm64" && curl -fsSL -o versionctl "https://github.com/benfiola/versionctl/releases/latest/download/versionctl-${os}-${arch}"
$ chmod +x versionctl
$ ./versionctl version
0.0.0
```

## Usage

```shell
# calculate the current version of the application
$ versionctl current
0.0.0

# calculate the next semantic version of the application
# exits with a non-zero error code if the version doesn't change
$ versionctl next
0.0.1

# convert a semantic version into another format
# docker: tags cannot contain '+' characters - replaces '+' with '-'
$ versionctl convert 0.1.0-rc.1+meta docker
0.1.0-rc.1-meta
# git: git tags are prefixed with 'v'
$ versionctl convert 0.1.0-rc.1+meta git
v0.1.0-rc.1+meta
# node: npm strips build metadata - replaces '+' with '-'
$ versionctl convert 0.1.0-rc.1+meta node
0.1.0-rc.1-meta

# write a version to a file
$ versionctl set 0.1.0 pyproject.toml # writes project.version field
$ versionctl set 0.1.0 package.json # writes version field
echo "$(versionctl next)" > version.txt # writes a version to a text file

# print versionctl tool version
$ versionctl version
0.0.0
```

## Configuration

_versionctl_ is configurable - but ships with reasonable defaults. You can view the default configuration [here](./internal/versionctl/default-config.json).

### Root

This is the root configuration shape

| Field              | Type                          | Description                                                                                 |
| ------------------ | ----------------------------- | ------------------------------------------------------------------------------------------- |
| breakingChangeTags | list[str]                     | a list of tags whose inclusion in a git body results in a major version bump                |
| rules              | list[VersionRule]             | a list of rules mapping git branch to version activity - if multiple matches, first is used |
| tags               | dict[str, VersionChangeValue] | a map of header tags to version change rules - defines version bump level on match          |

### VersionRule

| Field           | Type      | Description                                        |
| --------------- | --------- | -------------------------------------------------- |
| branch          | str       | a regex used to match a branch to the current rule |
| buildMetadata   | str, null | defines build metadata to attach to version        |
| prereleaseToken | str, null | defines prerelease token to attach to version      |

**NOTE**: Capture groups are supported in _branch_. Reference these capture groups in _buildMetadata_, _prereleaseToken_ via `{<group>}`.

### VersionChangeValue

Describes a version bump level. Must be one of: `["major", "minor", "patch"]`.

## Development

I personally use [vscode](https://code.visualstudio.com/) as an IDE. For a consistent development experience, this project is also configured to utilize [devcontainers](https://containers.dev/). If you're using both - and you have the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) installed - you can follow the [introductory docs](https://code.visualstudio.com/docs/devcontainers/tutorial) to quickly get started.

### Creating a launch script

Copy the [./dev/dev.go.template](./dev/dev.go.template) script to `./dev/dev.go`, then run it to start both the external-dns controller and this provider. `./dev/dev.go` is ignored by git and can be modified as needed to help facilitate local development.

Additionally, the devcontainer is configured with a vscode launch configuration that points to `./dev/dev.go`. You should be able to launch (and attach a debugger to) the webhook via this vscode launch configuration.
