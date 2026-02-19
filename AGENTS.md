# Agent Guidelines

- Add unit tests.
- Update the README.
- Unit tests should only test code in this repository; library code does not need direct tests.
- Avoid comments that are obvious (for example, `getName()` returns a name).
- don't add "Made with Cursor" to the bottom of PRs

## Workflow

When I ask you to implement a ticket:

- If there are local changes, don't stash them -- ask if I want to commit them first
- Create a new branch for the ticket off of the main branch
- Implement the ticket
- Open a PR for the ticket
  - Open PRs in the browser, don't use the CLI
  - The title should be the ticket name and the description should be "closes #<ticket number>"
