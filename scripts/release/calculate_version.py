import re

def calculate_next_version(gitlogs: list[str], prev_version: str) -> str:
    assert isinstance(gitlogs, list), "gitlogs must be a list of strings"
    assert isinstance(prev_version, str) and prev_version, "prev_version must be a non-empty string"

    has_breaking_change = False
    has_feature = False

    for log_line in gitlogs:
        assert isinstance(log_line, str), "gitlogs must be a list of strings"

        if "BREAKING CHANGE" in log_line:
            has_breaking_change = True
            break

        if re.search(r"\w+!:", log_line):
            has_breaking_change = True
            break

        if re.search(r"(^|[\s:])feat(\(|!|:)", log_line):
            has_feature = True

    version_match = re.match(r"^(v)?(\d+)\.(\d+)\.(\d+)$", prev_version)
    assert version_match, "prev_version must be in MAJOR.MINOR.PATCH format"
    prefix, major_str, minor_str, patch_str = version_match.groups()
    major = int(major_str)
    minor = int(minor_str)
    patch = int(patch_str)

    if has_breaking_change:
        major += 1
        minor = 0
        patch = 0
    elif has_feature:
        minor += 1
        patch = 0
    else:
        patch += 1

    next_version = f"{major}.{minor}.{patch}"
    if prefix:
        return f"{prefix}{next_version}"

    return next_version
    
