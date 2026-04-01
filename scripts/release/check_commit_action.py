import json
import os
import sys

from check_commit import check_commit_message 

def main() -> int:
    event_path = os.environ.get("GITHUB_EVENT_PATH")
    assert event_path, "GITHUB_EVENT_PATH is not set"

    with open(event_path, "r", encoding="utf-8") as event_file:
        event = json.load(event_file)

    pull_request = event.get("pull_request")
    assert pull_request, "pull_request payload is missing"

    title = pull_request.get("title")
    assert title, "pull_request.title is missing"

    if not check_commit_message(title):
        print(f'PR title is not conventional: "{title}"', file=sys.stderr)
        return 1

    print(f'PR title is conventional: "{title}"')
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
