// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * Test Setup
 *
 * This file is run before each test file.
 * Configure global test utilities, mocks, and matchers here.
 */

import '@testing-library/jest-dom';
import { afterAll, beforeAll } from 'vitest';

// Global test setup
beforeAll(() => {
  // Setup before all tests
});

afterAll(() => {
  // Cleanup after all tests
});

// Mock window.matchMedia for component tests
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});
