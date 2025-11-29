# Contributing to Code Pathfinder

## Introduction

Welcome to Code Pathfinder, the open-source alternative to CodeQL. Designed for precise flow analysis and advanced structural search, Code Pathfinder identifies vulnerabilities in your code. Currently optimized for Java, Code Pathfinder offers robust query support to enhance your code’s security and integrity.

## IDE Support

We recommend using the following IDEs for developing with Code Pathfinder:
- GoLand
- VS Code

## Core Project

The core of Code Pathfinder is the `sast-engine` project, written in Go.

## Build System

We use the Gradle build system to ensure seamless development across Linux, Windows, and macOS.

### Gradle Commands

Here are the key Gradle commands to work with Code Pathfinder:

- **Build the Binary**
  ```sh
  gradle buildGo
  ```

- **Run the Application**
  ```sh
  gradle runGo
  ```
  This will run the application with an example test directory source code.

- **Clean the Build Directory**
  ```sh
  gradle clean
  ```
  This command clears the build directory and binary.

- **Run Tests**
  ```sh
  gradle testGo
  ```
  This command runs tests on the source code.

- **Lint the Source Code**
  ```sh
  gradle lintGo
  ```
  This command is useful for linting the source code.

- **Prepare for Release**
  ```sh
  gradle prepareRelease
  ```
  This command helps in bumping the version and publishing the branch.

## Contribution Guidelines

We appreciate your contributions to Code Pathfinder! Here are some guidelines to help you get started:

1. **Fork the Repository**
   - Create a fork of the Code Pathfinder repository to your GitHub account.

2. **Clone the Repository**
   - Clone your fork to your local machine:
     ```sh
     git clone https://github.com/your-username/code-pathfinder.git
     ```

3. **Create a Branch**
   - Create a new branch for your feature or bug fix:
     ```sh
     git checkout -b feature-or-bugfix-branch
     ```

4. **Make Changes**
   - Implement your changes in the new branch.

5. **Run Tests**
   - Ensure that all tests pass:
     ```sh
     gradle testGo
     ```

6. **Commit Changes**
   - Commit your changes with a clear and concise message:
     ```sh
     git commit -m "Description of your changes"
     ```

7. **Push to GitHub**
   - Push your changes to your forked repository:
     ```sh
     git push origin feature-or-bugfix-branch
     ```

8. **Submit a Pull Request**
   - Open a pull request from your forked repository to the main repository.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

Thank you for contributing to Code Pathfinder!
