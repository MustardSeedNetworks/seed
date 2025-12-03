/**
 * Commitlint configuration
 *
 * Enforces conventional commit message format:
 * type(scope?): subject
 *
 * Examples:
 * - feat(dhcp): add phase timing breakdown
 * - fix(websocket): resolve connection drop
 * - docs: update installation instructions
 * - chore(deps): upgrade gopacket
 */
export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',     // New feature
        'fix',      // Bug fix
        'docs',     // Documentation changes
        'style',    // Code style changes (formatting)
        'refactor', // Code refactoring
        'perf',     // Performance improvements
        'test',     // Adding or updating tests
        'chore',    // Maintenance tasks
        'ci',       // CI/CD changes
        'build',    // Build system changes
        'revert',   // Revert a previous commit
      ],
    ],
    'scope-enum': [
      1,
      'always',
      [
        // Components
        'api',
        'auth',
        'capture',
        'config',
        'dhcp',
        'dns',
        'network',
        'websocket',
        // Cards
        'link',
        'switch',
        'vlan',
        'wifi',
        'cable',
        'gateway',
        // Frontend
        'ui',
        'components',
        'hooks',
        // Infrastructure
        'deps',
        'ci',
        'docker',
        'release',
      ],
    ],
    'subject-case': [2, 'never', ['start-case', 'pascal-case', 'upper-case']],
    'subject-empty': [2, 'never'],
    'subject-full-stop': [2, 'never', '.'],
    'type-case': [2, 'always', 'lower-case'],
    'type-empty': [2, 'never'],
  },
};
