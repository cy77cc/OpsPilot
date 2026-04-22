import tseslint from 'typescript-eslint';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';

const authScopeFiles = [
  'src/api/api.ts',
  'src/api/requestContext.ts',
  'src/components/Auth/AuthContext.tsx',
  'src/features/ai/api/chatApi.ts',
  'src/features/ai/api/assistApi.ts',
];

export default [
  {
    ignores: ['dist', 'node_modules'],
  },
  {
    files: ['**/*.{js,jsx,mjs,cjs}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    plugins: {
      'react-hooks': reactHooks,
    },
    rules: {},
  },
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parser: tseslint.parser,
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    plugins: {
      '@typescript-eslint': tseslint.plugin,
      'react-hooks': reactHooks,
    },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
    },
  },
  {
    files: authScopeFiles,
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'error',
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.object.name='localStorage'][callee.property.name='getItem']",
          message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
        },
        {
          selector: "CallExpression[callee.object.name='localStorage'][callee.property.name='setItem']",
          message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
        },
        {
          selector: "CallExpression[callee.object.name='localStorage'][callee.property.name='removeItem']",
          message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
        },
        {
          selector: "CallExpression[callee.object.object.name='window'][callee.object.property.name='localStorage'][callee.property.name='getItem']",
          message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
        },
        {
          selector: "CallExpression[callee.object.object.name='window'][callee.object.property.name='localStorage'][callee.property.name='setItem']",
          message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
        },
        {
          selector: "CallExpression[callee.object.object.name='window'][callee.object.property.name='localStorage'][callee.property.name='removeItem']",
          message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
        },
      ],
    },
  },
];
