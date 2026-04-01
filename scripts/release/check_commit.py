import re

def check_commit_message(title: str) -> bool:
    pattern = re.compile(
        r"^(revert: )?((feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([\w ._-]+\))?(!)?: .+)"
    )
    return bool(pattern.match(title))
