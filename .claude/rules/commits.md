# Making Git Commits

- Use conventional commits with a clear, descriptive message.
- Use the commit description for a bullet list of changes made.

# Creating Pull Requests

- After creating a PR, always check that CI passes. If CI fails, proactively diagnose and fix the failures without waiting to be asked.
- **The baseline for test failures is ALWAYS `main`, never the previous commit on the branch.** If a test passes on `main` but fails on your branch, it is a regression you introduced and you MUST fix it. Do not dismiss a failure just because a previous commit on the same branch also had it. The only tests you may ignore are those that are also failing or flaky on `main`.
- Common CI failure sources: broken tests from removed code paths, nil loggers in test setup, ESLint missing keys in Svelte `{#each}` blocks, and test selectors that are too broad.

# Merging Pull Requests

- Before merging a PR, first merge `origin/main` into the branch to ensure it's up-to-date.
- Run tests after merging to catch any integration issues before the final merge.
