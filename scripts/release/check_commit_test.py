import unittest

from scripts.release.check_commit import check_commit_message

class TestCheckCommitMessage(unittest.TestCase):
    def test_accepts_conventional_titles(self) -> None:
        titles = [
            "feat: add widget support",
            "fix(parser): handle nil tokens",
            "docs(readme): update install guide",
            "refactor!: remove legacy API",
            "chore(ci): bump actions",
            "revert: feat: add widget support",
            "perf(db layer): reduce allocations",
        ]

        for title in titles:
            with self.subTest(title=title):
                result = check_commit_message(title)

                self.assertTrue(result)

    def test_rejects_non_conventional_titles(self) -> None:
        titles = [
            "feature: add widget support",
            "feat add widget support",
            "feat(): add widget support",
            "feat: ",
            "docs(readme) update install guide",
            "",
        ]

        for title in titles:
            with self.subTest(title=title):
                result = check_commit_message(title)

                self.assertFalse(result)


if __name__ == "__main__":
    unittest.main()
