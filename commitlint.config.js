/**
 * @file Commitlint configuration
 * @description Enforces conventional commit format: type(scope?): subject
 *
 * @example
 * feat(snmp): add trap handling
 * fix(capture): resolve packet loss
 * docs: update API reference
 * chore(deps): upgrade dependencies
 */
export default {
  extends: ["@commitlint/config-conventional"],
  rules: {
    "type-enum": [
      2,
      "always",
      [
        "feat",     // New feature
        "fix",      // Bug fix
        "docs",     // Documentation changes
        "style",    // Code style changes (formatting)
        "refactor", // Code refactoring
        "perf",     // Performance improvements
        "test",     // Adding or updating tests
        "chore",    // Maintenance tasks
        "ci",       // CI/CD changes
        "build",    // Build system changes
        "revert",   // Revert a previous commit
      ],
    ],
    "scope-enum": [
      1,
      "always",
      [
        // Core components
        "api", "capture", "config", "daemon", "device", "snmp", "protocols", "storage", "templates",
        // Frontend
        "ui", "components", "hooks",
        // Infrastructure
        "deps", "ci", "docker", "release",
      ],
    ],
    "subject-case": [2, "never", ["start-case", "pascal-case", "upper-case"]],
    "subject-full-stop": [2, "never", "."],
    "subject-empty": [2, "never"],
    "type-case": [2, "always", "lower-case"],
    "type-empty": [2, "never"],
    "header-max-length": [2, "always", 100],
  },
};
