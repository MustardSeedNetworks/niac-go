/**
 * @file Commitlint configuration
 * @description Enforces conventional commit message format: type(scope?): subject
 *
 * @example
 * feat(snmp): add trap handling
 * fix(capture): resolve packet loss
 * docs: update API reference
 * chore(deps): upgrade dependencies
 */
module.exports = {
  extends: ["@commitlint/config-conventional"],
  rules: {
    "type-enum": [
      2,
      "always",
      [
        "feat", // New feature
        "fix", // Bug fix
        "docs", // Documentation changes
        "style", // Code style changes (formatting)
        "refactor", // Code refactoring
        "perf", // Performance improvements
        "test", // Adding or updating tests
        "chore", // Maintenance tasks
        "ci", // CI/CD changes
        "build", // Build system changes
        "revert", // Revert a previous commit
      ],
    ],
    "scope-enum": [
      1,
      "always",
      [
        // Core components
        "api",
        "capture",
        "config",
        "daemon",
        "device",
        "snmp",
        "protocols",
        "storage",
        "templates",
        // Frontend
        "ui",
        "components",
        "hooks",
        // Infrastructure
        "deps",
        "ci",
        "docker",
        "release",
      ],
    ],
    // Disallow subject lines in start-case, pascal-case, or upper-case
    "subject-case": [2, "never", ["start-case", "pascal-case", "upper-case"]],
    // Conventional commits do not end the subject line with a period
    "subject-full-stop": [2, "never", "."],
    "type-case": [2, "always", "lower-case"],
    "type-empty": [2, "never"],
  },
};
