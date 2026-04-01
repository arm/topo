import json
import os
import urllib.parse
import urllib.request
from typing import Optional

from scripts.release.calculate_version import (
    calculate_next_version,
)


def main() -> int:
    latest_version = fetch_latest_release_version()
    logs = fetch_logs_since(latest_version)
    next_version = calculate_next_version(logs, latest_version)
    output_file = os.environ.get("GITHUB_OUTPUT")
    if output_file:
        with open(output_file, "a", encoding="utf-8") as output:
            output.write(f"next_version={next_version}\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


def fetch_latest_release_version() -> str:
    api_url = "https://api.github.com/repos/arm/topo/releases/latest"
    request = urllib.request.Request(
        api_url,
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": "topo-release-script",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )

    github_token = os.environ.get("GITHUB_TOKEN")
    if github_token:
        request.add_header("Authorization", f"Bearer {github_token}")

    with urllib.request.urlopen(request, timeout=10) as response:
        payload = json.load(response)

    tag_name = payload.get("tag_name")
    assert isinstance(tag_name, str) and tag_name, "latest release tag is missing"
    return tag_name


def fetch_logs_since(tag: str) -> list[str]:
    assert isinstance(tag, str) and tag, "tag must be a non-empty string"

    repo = "arm/topo"
    api_base = f"https://api.github.com/repos/{repo}"

    repo_payload = _github_get_json(api_base)
    default_branch = repo_payload.get("default_branch")
    assert isinstance(default_branch, str) and default_branch, "default_branch is missing"

    compare_url = f"{api_base}/compare/{urllib.parse.quote(tag)}...{urllib.parse.quote(default_branch)}"
    logs: list[str] = []
    next_url = _add_query_params(compare_url, {"per_page": "100", "page": "1"})

    while next_url:
        payload, headers = _github_get_json_with_headers(next_url)
        commits = payload.get("commits", [])
        assert isinstance(commits, list), "compare response commits is not a list"

        for commit in commits:
            message = commit.get("commit", {}).get("message")
            if not isinstance(message, str) or not message:
                continue
            subject = message.splitlines()[0]
            if subject:
                logs.append(subject)

        next_url = _parse_link_header(headers.get("Link")).get("next")

    return logs


def _add_query_params(url: str, params: dict[str, str]) -> str:
    parsed = urllib.parse.urlparse(url)
    query = urllib.parse.parse_qs(parsed.query)
    for key, value in params.items():
        query[key] = [value]
    updated_query = urllib.parse.urlencode(query, doseq=True)
    return urllib.parse.urlunparse(parsed._replace(query=updated_query))


def _parse_link_header(link_header: Optional[str]) -> dict[str, str]:
    if not link_header:
        return {}

    links: dict[str, str] = {}
    for part in link_header.split(","):
        section = part.strip().split(";")
        if len(section) < 2:
            continue
        url_part = section[0].strip()
        rel_part = section[1].strip()
        if not (url_part.startswith("<") and url_part.endswith(">")):
            continue
        if not rel_part.startswith('rel="') or not rel_part.endswith('"'):
            continue
        url = url_part[1:-1]
        rel = rel_part[len('rel="') : -1]
        links[rel] = url
    return links


def _github_get_json(url: str) -> dict:
    payload, _headers = _github_get_json_with_headers(url)
    return payload


def _github_get_json_with_headers(url: str) -> tuple[dict, dict]:
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": "topo-release-script",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )

    github_token = os.environ.get("GITHUB_TOKEN")
    if github_token:
        request.add_header("Authorization", f"Bearer {github_token}")

    with urllib.request.urlopen(request, timeout=10) as response:
        payload = json.load(response)
        headers = dict(response.headers)

    assert isinstance(payload, dict), "GitHub API response is not a JSON object"
    return payload, headers
