# Contribution Manifest

- [Branches and Commits](#branches-and-commits)
- [Code and Structure](#code-and-structure)
- [Issues](#issues)
- [Pull Requests](#pull-requests)
- [Pre Commit](#pre-commit)

## Branches and Commits

### Structure

Branches:
`type/description-message`

Commits:
`type: description message`

### Types

| Type     | Example                         | Description                                                                          |
| -------- | ------------------------------- | ------------------------------------------------------------------------------------ |
| feat     | feat: add more workload types   | A new feature                                                                        |
| fix      | fix: stop tests from failing    | A fix of a bug                                                                       |
| chore    | chore: rm unneeded consts       | An annoying task, that has/had to be done                                            |
| refactor | refactor: improved error logs   | Changes that don't change any functionality                                          |
| revert   | revert: #5ff385d2               | A change that reverts a previous commit                                              |
| docs     | docs: added a readme            | Changes to documentation only                                                        |
| perf     | perf: sw to concurrent scanning | A change that only improves performance                                              |
| style    | style: rm leading spaces        | A change which makes the code look better without changing the code (eg. formatting) |

### Common Abbreviations

| Abbreviation | Meaning |
| ------------ | ------- |
| sw           | switch  |
| rm           | remove  |

## Code and Structure

Try to stick to golang best practices and standards. Eg.:

<!-- keep this list updated everytime someone opens a pr with best practice issues  -->

- Package structure standards: https://github.com/golang-standards/project-layout
- Use guard clauses if applicable
- Try to avoid using else. Most of the time these can be replaced by just placing the else block content directly after the if or by refactoring the if block to be an additional function.
- Comments on funcs/types (esp. public ones)
- Only make functions public if necessary

## Issues

If applicable use the issue template. This ensures a consistent structure which makes it easier to find important details.
Issues which aren't ready for processing, can be marked as a draft by writing "Draft: " infront of the issue name.

## Pull Requests

Use the Pull Request Template. This ensures a consistent structure which makes it easier to find important details.
Set yourself and any other collaborators as assignee.

## Pre Commit

It is recommended to install pre-commit. This insures that formatting is consistent, you don't commit to protected branches and you don't accidentally commit broken code or new functionality without changing the tests. The installation process is in the [README](README.md#setting-up-pre-commit)
