import functools
import importlib.metadata
import json
import logging
import pathlib
import re
from typing import Any, Literal

import pydantic
import toml

import versionctl

from .git import Git

logger = logging.getLogger(__name__)


VersionChangeValue = (
    Literal["major"]
    | Literal["minor"]
    | Literal["patch"]
    | Literal["prerelease"]
    | Literal["none"]
)


class BaseModel(pydantic.BaseModel):
    """
    Pydantic base model subclass that parses data by both field name and alias
    """

    model_config = {"populate_by_name": True}


class VersionRule(BaseModel):
    """
    Represents a version rule definition
    """

    # Used to match rule to branch - could be a literal or a regex
    branch: str
    # Versions produced by this rule add a pre-release version if defined
    prerelease_token: str | None = pydantic.Field(None, alias="prereleaseToken")
    # Versions produced by this rule add build metadata if defined
    build_metadata: str | None = pydantic.Field(None, alias="buildMetadata")

    def match(self, branch: str) -> re.Match:
        branch_re = re.compile(self.branch)
        match = branch_re.match(branch)
        if not match:
            raise ValueError(branch)
        return match


class Configuration(BaseModel):
    """
    Represents a configuration object used to define versioning behavior
    """

    # A list of tags that represent immediate 'version' bumps (and are expected to be in the commit body)
    breaking_change_tags: list[str] | None = pydantic.Field(
        None, alias="breakingChangeTags"
    )
    # A list of rules to match against branches
    rules: list[VersionRule]
    # A list of tags matched against commit headers to determine version bump behavior
    tags: dict[str, VersionChangeValue]


def operator_error(operator: str, left: Any, right: Any) -> Exception:
    """
    Convenience method to produce a common ValueError when a comparison operation is invalid.
    """
    return ValueError(
        f"Operator {operator} not supported for types {type(left)} and {type(right)}"
    )


@functools.total_ordering
class VersionChange:
    """
    Represents a version change - can be added to a 'Version' instance to produce a bumped version object.
    """

    value: VersionChangeValue

    def __init__(self, value: VersionChangeValue):
        self.value = value

    def __int__(self) -> int:
        """
        Gives a version change an 'int' value - useful for ordering
        """
        if self.value == "major":
            return 4
        elif self.value == "minor":
            return 3
        elif self.value == "patch":
            return 2
        elif self.value == "prerelease":
            return 1
        elif self.value == "none":
            return 0
        raise NotImplementedError(self.value)

    def __lt__(self, other: Any) -> bool:
        """
        Used in conjunction with `functools.total_ordering` to allow comparison operations with this class.
        """
        if not isinstance(other, VersionChange):
            raise operator_error("<", self, other)
        return int(self) < int(other)

    def __str__(self) -> str:
        return self.value


class PrereleaseVersionChange(VersionChange):
    """
    Specifically represents a 'prerelease' version change - that additionally requires a prerelease token.
    """

    token: str

    def __init__(self, token: str):
        super().__init__("prerelease")
        self.token = token


@functools.total_ordering
class Maximum:
    """
    Helper class that represents a 'maximum' value.

    This allows us to sort non-prerelease versions as 'greater than' prerelease versions.
    """

    def __lt__(self, other: Any) -> bool:
        if isinstance(other, Maximum):
            return False
        return True


semver_re = re.compile(
    r"(?P<major>\d+)"
    r"\.(?P<minor>\d+)"
    r"\.(?P<patch>\d+)"
    r"(?:-(?P<prerelease_tag>.+)\.(?P<prerelease_count>\d+))?"
    r"(?:\+(?P<build_metadata>.*))?"
)

Prerelease = tuple[str, int] | None
BuildMetadata = str | None
VersionFormat = Literal["docker"] | Literal["git"] | Literal["semver"] | Literal["node"]


@functools.total_ordering
class Version:
    """
    Represents a version - each attribute representing a component of the version.
    """

    major: int
    minor: int
    patch: int
    prerelease: Prerelease
    build_metadata: BuildMetadata

    def __init__(
        self,
        major: int,
        minor: int,
        patch: int,
        prerelease: Prerelease = None,
        build_metadata: BuildMetadata = None,
    ):
        self.major = major
        self.minor = minor
        self.patch = patch
        self.prerelease = prerelease
        self.build_metadata = build_metadata

    def update(self, **kwargs) -> "Version":
        """
        Produces a new version, updating existing attributes with values found in 'kwargs'.
        """
        data = dict(self.__dict__)
        data.update(**kwargs)
        return type(self)(**data)

    def to_release(self) -> "Version":
        """
        Returns a version stripped of prerelease versions or build metadata
        """
        return Version(major=self.major, minor=self.minor, patch=self.patch)

    def to_str(self, format: VersionFormat = "semver") -> str:
        """
        Converts the version into a string
        """
        if format == "docker":
            semver = self.to_str(format="semver")
            version = semver.replace("+", "-")
            return version
        elif format == "git":
            semver = self.to_str(format="semver")
            version = f"v{semver}"
            return version
        elif format == "node":
            semver = self.to_str(format="semver")
            version = semver.replace("+", "-")
            return version
        elif format == "semver":
            version = f"{self.major}.{self.minor}.{self.patch}"
            if self.prerelease:
                version = f"{version}-{self.prerelease[0]}.{self.prerelease[1]}"
            if self.build_metadata:
                version = f"{version}+{self.build_metadata}"
            return version
        else:
            raise NotImplementedError(format)

    def __add__(self, other: Any) -> "Version":
        """
        Enables the "+" operation for Version objects with VersionChange objects.

        Performs a version bump, and returns a new version object.
        """
        if not isinstance(other, VersionChange):
            raise operator_error("+", self, other)
        new_version = Version(**self.__dict__)

        if other.value == "major":
            new_version.major += 1
            new_version.minor = 0
            new_version.patch = 0
            new_version.prerelease = None
            new_version.build_metadata = None
        elif other.value == "minor":
            new_version.minor += 1
            new_version.patch = 0
            new_version.prerelease = None
            new_version.build_metadata = None
        elif other.value == "patch":
            new_version.patch += 1
            new_version.prerelease = None
            new_version.build_metadata = None
        elif isinstance(other, PrereleaseVersionChange):
            if new_version.prerelease and new_version.prerelease[0] == other.token:
                new_version.prerelease = (
                    new_version.prerelease[0],
                    new_version.prerelease[1] + 1,
                )
            else:
                new_version.prerelease = (other.token, 1)
        else:
            raise NotImplementedError(other)

        return new_version

    def __lt__(self, other: Any) -> bool:
        """
        Used in conjunction with `functools.total_ordering` to enable all comparison operations on this object.
        """
        if not isinstance(other, Version):
            raise operator_error("<", self, other)
        max = Maximum()
        left = (self.major, self.minor, self.patch, self.prerelease or max)
        right = (other.major, other.minor, other.patch, other.prerelease or max)
        return left < right

    def __str__(self) -> str:
        """
        Returns a string representation of this version
        """
        return self.to_str()

    def __sub__(self, other: Any) -> "VersionChange":
        """
        Enables the "-" operation for Version objects with other Version objects.

        Returns a VersionChange object representing the largest difference between versions.
        """
        if not isinstance(other, Version):
            raise operator_error("-", type(self), type(other))
        if self.major != other.major:
            return VersionChange(value="major")
        if self.minor != other.minor:
            return VersionChange(value="minor")
        if self.patch != other.patch:
            return VersionChange(value="patch")
        return VersionChange(value="none")

    @classmethod
    def from_semver(cls, semver: str) -> "Version":
        """
        Creates a version object from a string
        """
        match = semver_re.match(semver)
        if not match:
            raise ValueError(semver)
        data = match.groupdict()
        major = int(data["major"])
        minor = int(data["minor"])
        patch = int(data["patch"])
        prerelease = None
        if data.get("prerelease_tag") and data.get("prerelease_count"):
            prerelease = data["prerelease_tag"], int(data["prerelease_count"])
        build_metadata = data.get("build_metadata")
        return cls(
            major=major,
            minor=minor,
            patch=patch,
            prerelease=prerelease,
            build_metadata=build_metadata,
        )

    @classmethod
    def from_git_tag(cls, tag: str) -> "Version":
        """
        Parses a git tag into a version object
        """
        if not tag.startswith("v"):
            raise ValueError(tag)
        tag = tag[1:]
        return cls.from_semver(tag)


class Parser:
    """
    Parses commit messages to obtain version and version bump behavior
    """

    breaking_change_tags: list[str] | None
    tags: dict[str, VersionChangeValue]

    def __init__(
        self,
        *,
        breaking_change_tags: list[str] | None,
        tags: dict[str, VersionChangeValue],
    ):
        self.breaking_change_tags = breaking_change_tags
        self.tags = tags

    def parse(self, message: str) -> VersionChangeValue:
        """
        Parses the message and returns the type of version bump to be performed.
        """
        lines = message.splitlines()
        if not lines:
            return "none"

        # parse header
        change = None
        for tag, current_change in self.tags.items():
            if not lines[0].startswith(tag):
                continue
            change = current_change
            break
        if not change:
            return "none"
        if not self.breaking_change_tags:
            return change

        # parse body
        for line in lines[1:]:
            for tag in self.breaking_change_tags:
                if not line.startswith(tag):
                    continue
                return "major"
        return change


sanitize_re = re.compile("[^a-zA-Z0-9.]+")


class Analyzer:
    """
    Analyzes the local working copy to infer version data
    """

    git: Git
    parser: Parser
    rules: list[VersionRule]

    def __init__(self, *, git: Git, parser: Parser, rules: list[VersionRule]):
        self.git = git
        self.parser = parser
        self.rules = rules

    def match_version_rule(self, branch: str) -> VersionRule | None:
        """
        Matches a branch to one of the configured version rules.

        Returns None if no rules match.
        """
        for rule in self.rules:
            try:
                rule.match(branch)
                return rule
            except ValueError:
                continue
        return None

    def get_repo_data(self) -> Version:
        """
        Returns repo-wide data.

        This includes the max version from all known tags.
        """
        repo_version = Version.from_semver("0.0.0")
        for tag in self.git.list_tags():
            try:
                version = Version.from_git_tag(tag)
            except ValueError:
                continue
            repo_version = max(repo_version, version)
        return repo_version

    def get_ancestral_data(self) -> tuple[Version, VersionChange]:
        """
        Returns data from the current commit's ancestry.

        This includes the max ancestral release.
        This includes the largest unversioned change.
        """
        change = VersionChange(value="none")
        release = None
        for commit in self.git.iter_commits():
            for tag in commit.tags:
                try:
                    version = Version.from_git_tag(tag)
                except ValueError:
                    continue
                if version.prerelease:
                    continue
                release = max(release or version, version)
            if release:
                logger.debug(f"commit {commit.commit_hash}: ancestor release {release}")
                break
            commit_change = VersionChange(self.parser.parse(commit.message))
            logger.debug(f"commit {commit.commit_hash}: change {commit_change}")
            change = max(change, commit_change)
        release = release or Version.from_semver("0.0.0")
        return release, change

    def get_current_version(self) -> Version:
        """
        Determines the current version based upon repo data.

        Returns a version object
        """
        repo_version = self.get_repo_data()
        logger.info(f"repo version: {repo_version}")

        return repo_version

    def get_next_version(self) -> Version:
        """
        Determines the next version based upon repo and commit history data.

        Returns a version object.
        """
        branch = self.git.get_current_branch()
        logger.info(f"branch: {branch}")

        rule = self.match_version_rule(branch)
        if not rule:
            raise ValueError(f"no rules match: {branch}")
        logger.info(f"rule: {rule.branch}")

        format_data = {**rule.match(branch).groupdict(), "branch": branch}
        prerelease_token = rule.prerelease_token
        if prerelease_token:
            prerelease_token = prerelease_token.format(**format_data)
        logger.info(f"prerelease token: {prerelease_token}")

        build_metadata = rule.build_metadata
        if build_metadata:
            build_metadata = build_metadata.format(**format_data)
        logger.info(f"build metadata: {build_metadata}")

        repo_version = self.get_repo_data()
        logger.info(f"repo version: {repo_version}")

        ancestor_release, change = self.get_ancestral_data()
        logger.info(f"ancestor release: {ancestor_release}")
        logger.info(f"change: {change}")

        if change == "none":
            raise ValueError(f"version unchanged")

        drift = repo_version - ancestor_release
        logger.info(f"drift: {drift}")

        if prerelease_token is not None:
            if drift < change:
                version = repo_version + change
            else:
                version = repo_version
            version += PrereleaseVersionChange(token=prerelease_token)
        else:
            if not repo_version.prerelease:
                version = repo_version + change
            else:
                if drift < change:
                    version = repo_version + change
                else:
                    version = repo_version.to_release()
        if build_metadata:
            version = version.update(build_metadata=build_metadata)
        logger.info(f"version: {version}")

        return version

    @classmethod
    def create(cls, config: Configuration) -> "Analyzer":
        """
        Creates an analyzer from a configuration object.
        """
        git = Git()
        parser = Parser(
            breaking_change_tags=config.breaking_change_tags, tags=config.tags
        )
        analyzer = Analyzer(git=git, parser=parser, rules=config.rules)
        return analyzer


def set_version(version: str, file: pathlib.Path):
    """
    Writes version into known file types
    """
    if file.name == "pyproject.toml":
        data = toml.loads(file.read_text())
        data["project"]["version"] = version
        file.write_text(toml.dumps(data))
    elif file.name == "package.json":
        data = json.loads(file.read_text())
        data["version"] = version
        file.write_text(json.dumps(data, indent=2))
    else:
        raise NotImplementedError(file)


def get_tool_version() -> str:
    """
    Gets the current version of the versionctl tool
    """
    metadata: dict[str, Any] = importlib.metadata.metadata(versionctl.__name__).json
    version = metadata.get("version")
    if not isinstance(version, str):
        raise RuntimeError(f"version not found")
    return version
