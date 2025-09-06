/**
 * This file provides a mapping from application types to prompt files.
 * It helps the security analyzer load the appropriate prompts based on the detected application type.
 */

const promptMappings = [
  // Web Applications
  {
    category: 'web',
    promptPath: 'web/generic.txt'
  },
  {
    category: 'web',
    subcategory: 'nodejs',
    technology: 'express',
    promptPath: 'web/nodejs/express.txt'
  },
  {
    category: 'web',
    subcategory: 'python',
    technology: 'django',
    promptPath: 'web/python/django-flask-fastapi.txt'
  },
  {
    category: 'web',
    subcategory: 'python',
    technology: 'flask',
    promptPath: 'web/python/django-flask-fastapi.txt'
  },
  {
    category: 'web',
    subcategory: 'python',
    technology: 'fastapi',
    promptPath: 'web/python/django-flask-fastapi.txt'
  },

  // Mobile Applications
  {
    category: 'mobile',
    subcategory: 'android',
    promptPath: 'mobile/android/android.txt'
  },
  {
    category: 'mobile',
    subcategory: 'ios',
    promptPath: 'mobile/ios/ios.txt'
  },
  {
    category: 'mobile',
    subcategory: 'cross-platform',
    technology: 'react-native',
    promptPath: 'mobile/cross-platform/react-native.txt'
  },
  {
    category: 'mobile',
    subcategory: 'cross-platform',
    technology: 'flutter',
    promptPath: 'mobile/cross-platform/flutter.txt'
  },

  // Backend Services
  {
    category: 'backend',
    subcategory: 'http',
    promptPath: 'backend/http/rest-api.txt'
  },
  {
    category: 'backend',
    subcategory: 'graphql',
    promptPath: 'backend/graphql/graphql-api.txt'
  },
  {
    category: 'backend',
    subcategory: 'grpc',
    promptPath: 'backend/grpc/grpc-api.txt'
  },
  {
    category: 'backend',
    subcategory: 'serverless',
    promptPath: 'backend/serverless/serverless-functions.txt'
  },

  // Frontend Applications
  {
    category: 'frontend',
    subcategory: 'react',
    promptPath: 'frontend/react/react-app.txt'
  },
  {
    category: 'frontend',
    subcategory: 'vue',
    promptPath: 'frontend/vue/vue-app.txt'
  },

  // Desktop Applications
  {
    category: 'desktop',
    subcategory: 'electron',
    promptPath: 'desktop/electron/electron-app.txt'
  },

  // Libraries
  {
    category: 'library',
    subcategory: 'node',
    promptPath: 'library/node/node-library.txt'
  },

  // CLI Applications
  {
    category: 'cli',
    subcategory: 'node',
    promptPath: 'cli/node/node-cli.txt'
  },
  {
    category: 'cli',
    subcategory: 'python',
    promptPath: 'cli/python/python-cli.txt'
  }
];

/**
 * Get the prompt path based on application profile
 * @param {string} category Main application category
 * @param {string} subcategory Optional subcategory
 * @param {string} technology Optional specific technology
 * @returns {string} The path to the most specific matching prompt
 */
function getPromptPath(category, subcategory, technology) {
  // First try to find an exact match with all criteria
  let match = promptMappings.find(
    (mapping) =>
      mapping.category === category &&
      mapping.subcategory === subcategory &&
      mapping.technology === technology
  );

  // If no exact match, try with just category and subcategory
  if (!match) {
    match = promptMappings.find(
      (mapping) =>
        mapping.category === category &&
        mapping.subcategory === subcategory &&
        !mapping.technology
    );
  }

  // If still no match, fall back to just category
  if (!match) {
    match = promptMappings.find(
      (mapping) =>
        mapping.category === category &&
        !mapping.subcategory &&
        !mapping.technology
    );
  }

  // If nothing matches, return the threat modeling prompt as fallback
  return match ? match.promptPath : 'common/threat-modeling.txt';
}

/**
 * Get the application profiler prompt
 * @returns {string} The path to the application profiler prompt
 */
function getAppProfilerPrompt() {
  return 'common/app-profiler.txt';
}

module.exports = {
  getPromptPath,
  getAppProfilerPrompt
};
