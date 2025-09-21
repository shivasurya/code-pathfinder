#!/usr/bin/env node

/**
 * Simple test for DefectDojo formatter
 * Run with: node test/defectdojo-formatter-test.js
 */

const { DefectDojoFormatter } = require('../lib/formatters/defectdojo-formatter');

// Sample SecureFlow scan result for testing
const sampleScanResult = {
  timestamp: new Date().toISOString(),
  projectPath: '/path/to/test/project',
  model: 'claude-3-5-sonnet-20241022',
  totalFiles: 25,
  filesAnalyzed: 15,
  iterations: 3,
  issues: [
    {
      title: 'SQL Injection vulnerability in user.js:45',
      severity: 'Critical',
      description: 'The application constructs SQL queries using string concatenation without proper input validation. This could allow an attacker to execute arbitrary SQL commands. Found in file user.js at line 45.',
      recommendation: 'Use parameterized queries or prepared statements to prevent SQL injection attacks. Validate and sanitize all user inputs before using them in database queries.'
    },
    {
      title: 'Cross-Site Scripting (XSS) in template rendering',
      severity: 'High',
      description: 'User input is directly rendered in HTML templates without proper escaping, potentially allowing XSS attacks. CWE-79 vulnerability detected.',
      recommendation: 'Implement proper output encoding and use template engines with automatic escaping enabled.'
    },
    {
      title: 'Hardcoded API key in config.py',
      severity: 'Medium',
      description: 'API key is hardcoded in the source code at config.py line 12, which poses a security risk if the code is exposed.',
      recommendation: 'Move sensitive credentials to environment variables or secure configuration files that are not committed to version control.'
    },
    {
      title: 'Weak password policy',
      severity: 'Low',
      description: 'The application does not enforce strong password requirements, making it vulnerable to brute force attacks.',
      recommendation: 'Implement strong password policies including minimum length, complexity requirements, and account lockout mechanisms.'
    }
  ],
  summary: {
    critical: 1,
    high: 1,
    medium: 1,
    low: 1
  }
};

console.log('🧪 Testing DefectDojo Formatter...\n');

try {
  // Test formatting
  console.log('1. Testing formatFindings()...');
  const defectDojoFindings = DefectDojoFormatter.formatFindings(sampleScanResult);
  console.log('✅ Formatting successful');
  
  // Test validation
  console.log('2. Testing validateFindings()...');
  const validation = DefectDojoFormatter.validateFindings(defectDojoFindings);
  if (validation.valid) {
    console.log('✅ Validation passed');
  } else {
    console.log('❌ Validation failed:');
    validation.errors.forEach(error => console.log(`   ${error}`));
    process.exit(1);
  }
  
  // Display results
  console.log('\n📊 Test Results:');
  console.log(`   Original issues: ${sampleScanResult.issues.length}`);
  console.log(`   DefectDojo findings: ${defectDojoFindings.findings.length}`);
  
  console.log('\n🔍 Sample DefectDojo Finding:');
  const sampleFinding = defectDojoFindings.findings[0];
  console.log(JSON.stringify(sampleFinding, null, 2));
  
  console.log('\n📈 Severity Distribution:');
  const severityCounts = {
    Critical: defectDojoFindings.findings.filter(f => f.severity === 'Critical').length,
    High: defectDojoFindings.findings.filter(f => f.severity === 'High').length,
    Medium: defectDojoFindings.findings.filter(f => f.severity === 'Medium').length,
    Low: defectDojoFindings.findings.filter(f => f.severity === 'Low').length,
    Info: defectDojoFindings.findings.filter(f => f.severity === 'Info').length
  };
  
  Object.entries(severityCounts).forEach(([severity, count]) => {
    if (count > 0) {
      console.log(`   ${severity}: ${count}`);
    }
  });
  
  console.log('\n🏷️  Sample Tags:');
  const allTags = new Set();
  defectDojoFindings.findings.forEach(finding => {
    if (finding.tags) {
      finding.tags.forEach(tag => allTags.add(tag));
    }
  });
  console.log(`   ${Array.from(allTags).join(', ')}`);
  
  console.log('\n✅ All tests passed! DefectDojo formatter is working correctly.');
  
} catch (error) {
  console.error('❌ Test failed:', error.message);
  if (process.env.DEBUG) {
    console.error(error.stack);
  }
  process.exit(1);
}
