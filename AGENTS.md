# Agent Guidelines

- Avoid comments that are obvious (for example, `getName()` returns a name).
- Update the README.
- Add unit tests.
  - Unit tests should only test code in this repository; library code does not need direct tests.
  - Keep unit tests concise and focused: each test file should only test functions defined in the corresponding source file (e.g., `output_test.go` tests functions from `output.go`).
  - Avoid duplicating tests across files. If a function is already tested in its own test file, don't re-test it elsewhere with different input data.
  - Use mocks where appropriate to isolate the unit under test.

## Workflow

When I ask you to implement a ticket:

- If there are local changes, don't stash them -- ask if I want to commit them first
- Create a new branch for the ticket off of the main branch
- Implement the ticket
- Open a PR for the ticket
  - Open PRs in the browser, don't use the CLI
  - The title should be the ticket name and the description should be "closes #<ticket number>"
  - don't add "Made with Cursor" to the bottom of PRs
