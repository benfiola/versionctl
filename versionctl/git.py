import dataclasses
from typing import Iterator

from .command import run_command


@dataclasses.dataclass
class Commit:
    """
    Represents commit data parsed from the git cli
    """

    commit_hash: str
    message: str
    tags: list[str]


class Git:
    """
    Implements a client for the git command-line utility.
    """

    def get_current_branch(self) -> str:
        """
        Gets the current branch from the local working copy
        """
        branch = run_command(["git", "symbolic-ref", "--short", "HEAD"]).strip()
        return branch

    def iter_commits(self, head: str | None = None) -> Iterator[Commit]:
        """
        Iterates over commits in order from the provided head.

        If 'head' is None, uses 'HEAD'.
        """
        head = head or "HEAD"

        def _get_commit(_skip: int = 0):
            _hashes = (
                run_command(
                    ["git", "rev-list", head, "--max-count=1", f"--skip={_skip}"]
                )
                .strip()
                .splitlines()
            )
            _hash = _hashes[0] if _hashes else None
            return _hash

        def _get_tags(_hash: str) -> list[str]:
            tags = (
                run_command(["git", "tag", "--points-at", _hash]).strip().splitlines()
            )
            return tags

        def _get_message(_hash: str) -> str:
            message = (
                run_command(
                    [
                        "git",
                        "rev-list",
                        _hash,
                        "--format=%B",
                        "--max-count=1",
                    ]
                )
                .strip()
                .splitlines()[1]
            )
            return message

        skip = 0
        commit_hash = _get_commit()
        while commit_hash is not None:
            skip += 1
            tags = _get_tags(commit_hash)
            message = _get_message(commit_hash)
            commit = Commit(commit_hash=commit_hash, message=message, tags=tags)
            yield commit
            commit_hash = _get_commit(skip)

    def list_tags(self) -> list[str]:
        """
        Lists all tags known to the repo.
        """
        tags = run_command(["git", "tag"]).strip().splitlines()
        return tags
