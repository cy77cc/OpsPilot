import tseslint from 'typescript-eslint';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';

const authScopeFiles = [
  'src/api/api.ts',
  'src/api/requestContext.ts',
  'src/api/modules/auth.ts',
  'src/app/session/sessionStore.ts',
  'src/app/scope/scopeStore.ts',
  'src/app/scope/useScope.ts',
  'src/components/Auth/AuthContext.tsx',
  'src/features/ai/api/chatApi.ts',
  'src/features/ai/api/assistApi.ts',
  'src/hooks/useNotificationWebSocket.ts',
  'src/utils/tokenManager.ts',
];

const localStorageRestrictedFiles = [
  'src/api/api.ts',
  'src/api/requestContext.ts',
  'src/api/modules/auth.ts',
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
    rules: {
      eqeqeq: ['error', 'always'],
      curly: ['error', 'all'],
      'no-var': 'error',
      'prefer-const': 'error',
      'no-debugger': 'error',
    },
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
      eqeqeq: ['error', 'always'],
      curly: ['error', 'all'],
      'no-var': 'error',
      'prefer-const': 'error',
      'no-debugger': 'error',
      '@typescript-eslint/consistent-type-imports': ['warn', { prefer: 'type-imports' }],
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
    },
  },
  {
    files: authScopeFiles,
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'error',
    },
  },
  {
    files: localStorageRestrictedFiles,
    rules: {
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
