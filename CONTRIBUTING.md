## Before contributing

This guideline explains how to contribute to Manboster. You **MUST** read this before writing or refactoring any code.

When contributing to this project (via `git push` or Pull Request), these guidelines apply, and you are held responsible for the code you commit.

## 1. About the AI code

We accept the use of AI-assisted or AI-generated code. While AI is fast and capable, we still expect that:  

- 1. You should understand at least half or all of the logic, and be aware of how AI-written code may impact the system.  

- 2. Follow our coding standards.

- 3. For PRs initiated by an AI Agent, we **must know** who **instructed** the AI Agent to initiate it.

## 2. About the commit

Commit messages should be concise and clear, with the first letter capitalized, and the title should not exceed 50 characters.

If a body is needed, be sure to explain the "what," "why," and "how." The body should not exceed 200 characters.


Please note, do not use [Conventional Commits](https://www.conventionalcommits.org/) as the standard for commit messages.

## 3. About the issue

Issues and features should be documented and archived in the form of an issue.

Before initiating a major change, please open an issue first, and then link to that issue in the pull request.


## 5. About the style

You should run `go fmt` before you commit your code.

Please follow standard Go naming conventions: use `MixedCaps` (PascalCase) for exported identifiers and `camelCase` for unexported variables.

Please use human-friendly function, variable, type names. Meaningless or ambiguous words **ARE NOT** allowed in any identifiers.

Comments in code **MUST BE** English, not other languages.


## 6. About the testing

You **MUST** test your changes before contributing.

Please run the following commands in order for testing:

- go build -o minibp cmd/minibp/main.go
- ./minibp -a
- ninja
- cd examples && ../minibp -a && ninja

Please ensure that Java, GCC, G++, and Ninja-build are installed on the system.
