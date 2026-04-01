import unittest

from scripts.release.calculate_version import calculate_next_version


class TestCalculateNextVersion(unittest.TestCase):
    def test_returns_next_major_when_breaking_change_present(self) -> None:
        gitlogs = [
            "fix: correct typo in output",
            "feat!: remove deprecated flags",
            "chore: update deps",
        ]

        result = calculate_next_version(gitlogs, "1.2.3")

        self.assertEqual(result, "2.0.0")

    def test_returns_next_major_when_breaking_change_footer_present(self) -> None:
        gitlogs = [
            "feat: add cache support",
            "docs: update changelog",
            "refactor: tidy modules BREAKING CHANGE",
        ]

        result = calculate_next_version(gitlogs, "0.9.9")

        self.assertEqual(result, "1.0.0")

    def test_returns_next_minor_when_feature_present_without_breaking(self) -> None:
        gitlogs = [
            "fix: handle nil pointers",
            "feat(parser): accept new syntax",
            "chore: bump tooling",
        ]

        result = calculate_next_version(gitlogs, "1.2.3")

        self.assertEqual(result, "1.3.0")

    def test_returns_next_bugfix_when_no_feature_or_breaking(self) -> None:
        gitlogs = [
            "fix: avoid panic on empty input",
            "docs: clarify usage",
            "chore: update dependencies",
        ]

        result = calculate_next_version(gitlogs, "1.2.3")

        self.assertEqual(result, "1.2.4")

    def test_preserves_v_prefix(self) -> None:
        gitlogs = [
            "fix: avoid panic on empty input",
        ]

        result = calculate_next_version(gitlogs, "v2.3.4")

        self.assertEqual(result, "v2.3.5")


if __name__ == "__main__":
    unittest.main()
