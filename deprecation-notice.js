#!/usr/bin/env node

const colors = {
    yellow: '\x1b[33m',
    red: '\x1b[31m',
    green: '\x1b[32m',
    cyan: '\x1b[36m',
    reset: '\x1b[0m',
    bold: '\x1b[1m',
    dim: '\x1b[2m'
};

console.log(`
${colors.yellow}${colors.bold}+----------------------------------------------------------------+
|                                                                |
|                    DEPRECATION NOTICE                          |
|                                                                |
+----------------------------------------------------------------+${colors.reset}

The ${colors.bold}codepathfinder${colors.reset} npm package is ${colors.red}DEPRECATED${colors.reset}.

${colors.green}Please migrate to pip:${colors.reset}
    ${colors.cyan}${colors.bold}pip install codepathfinder${colors.reset}

${colors.green}Why?${colors.reset}
    ${colors.dim}- pip installation includes BOTH the CLI binary AND Python DSL
    - Write and run custom security rules with one installation
    - Better cross-platform support via platform wheels
    - Single version to manage${colors.reset}

${colors.green}Migration steps:${colors.reset}
    ${colors.dim}1.${colors.reset} npm uninstall -g codepathfinder
    ${colors.dim}2.${colors.reset} pip install codepathfinder
    ${colors.dim}3.${colors.reset} pathfinder --version  ${colors.dim}# verify installation${colors.reset}

${colors.green}Documentation:${colors.reset}
    ${colors.cyan}https://codepathfinder.dev/install${colors.reset}

${colors.yellow}The binary will still be installed for backward compatibility.${colors.reset}
${colors.yellow}Future updates will only be available via pip.${colors.reset}
`);
