# Contribution Manifest

- [Branches and Commits](#branches-and-commits)
- [Code and Structure](#code-and-structure)
- [Issues](#issues)
- [Pull Requests](#pull-requests)
- [Pre Commit](#pre-commit)
- [Versioning](#versioning)

## Branches and Commits

Branch and Commit names follow the [conventional commits specification](https://www.conventionalcommits.org/en/v1.0.0/)

### Structure

Branches:
`type/description-message`

Commits:
`type: description message`

### Types

| Type     | Example                         | Description                                                  |
| -------- | ------------------------------- | ------------------------------------------------------------ |
| feat     | feat: add more workload types   | A new feature                                                |
| fix      | fix: stop tests from failing    | A fix of a bug                                               |
| chore    | chore: rm unneeded consts       | An annoying task, that has/had to be done                    |
| refactor | refactor: improved error logs   | Changes that don't change any functionality                  |
| revert   | revert: #5ff385d2               | A change that reverts a previous commit                      |
| docs     | docs: added a readme            | Changes to documentation only                                |
| perf     | perf: sw to concurrent scanning | A change that only improves performance                      |
| style    | style: rm leading spaces        | A change which makes the code look better (formatting, etc.) |

## Code and Structure

Try to stick to golang best practices and standards:

<!-- keep this list updated every time someone opens a pr with best practice issues  -->

- [Package structure standards](https://github.com/golang-standards/project-layout)
- Use guard clauses if applicable
- Try to avoid using else.
  Most of the time these can be replaced by
  just placing the else block content directly after the if or
  by refactoring the if block to be an additional function.
- Comments on funcs/types (esp. public ones)
- Only make functions public if necessary

## Issues

If applicable use the issue template.
This ensures a consistent structure which makes it easier to find important details.
Issues which aren't ready for processing, can be marked as a draft by writing "Draft: " in front of the issue name.

## Pull Requests

Use the Pull Request Template.
This ensures a consistent structure which makes it easier to find important details.
Set yourself and any other collaborators as assignee.

## Pre Commit

It is recommended to install pre-commit.
This ensures that
formatting is consistent,
you don't commit to protected branches and
you don't accidentally commit broken code or new functionality without changing the tests.
The installation process is on [our website](https://caas-team.github.io/GoKubeDownscaler/guides/developing#setting-up-pre-commit).

## Versioning

New releases are automatically created when the appVersion in the Chart.yaml is updated in the main branch.
To merge a pull request which when merged would result in a new release,
the [`new release` label](https://github.com/caas-team/GoKubeDownscaler/labels/new%20release) has to be set on the PR.

Releases are following the semver versioning standard:

Layout: `<Major>.<Minor>.<Patch>` (example: 1.1.0)

- MAJOR: increment on breaking changes
- MINOR: increment on new functionality/features
- PATCH: increment on small bug fixes

You can find more information on semantic versioning [on the official website](https://semver.org/).
