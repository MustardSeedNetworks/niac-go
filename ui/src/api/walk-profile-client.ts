import { requestJsonCamelCase } from './requestCore';
import type { ScenarioDeviceProfile } from './scenario-client';
import type { WalkAnalysis } from './walk-analyze-types';

const captureResponseBufferMs = 5_000;

export interface WalkProfileReview {
  walkName: string;
  profile: ScenarioDeviceProfile;
  analysis: WalkAnalysis;
}

export interface WalkCaptureCredentials {
  target: string;
  port: number;
  version: '2c' | '3';
  community?: string;
  username?: string;
  authProtocol?: string;
  authPassword?: string;
  privProtocol?: string;
  privPassword?: string;
  timeoutSeconds: number;
}

export interface CapturedProfileCreateRequest {
  role: string;
  deviceType: string;
  vendor: string;
  model: string;
  platform: string;
  software: string;
  walkName: string;
}

export const importWalkProfile = (name: string, content: string, signal?: AbortSignal) =>
  requestJsonCamelCase<WalkProfileReview>(
    '/api/v1/walk/import',
    { name, content },
    { method: 'POST', signal },
    { maxRetries: 0, baseDelay: 0 },
  );

export const captureWalkProfile = (
  name: string,
  capture: WalkCaptureCredentials,
  signal?: AbortSignal,
) =>
  requestJsonCamelCase<WalkProfileReview>(
    '/api/v1/walk/capture-profile',
    { name, capture },
    { method: 'POST', signal },
    { maxRetries: 0, baseDelay: 0 },
    capture.timeoutSeconds * 1_000 + captureResponseBufferMs,
  );

export const createCapturedProfile = (profile: CapturedProfileCreateRequest) =>
  requestJsonCamelCase<ScenarioDeviceProfile>(
    '/api/v1/scenario/profiles/captured',
    profile,
    { method: 'POST' },
    { maxRetries: 0, baseDelay: 0 },
  );
