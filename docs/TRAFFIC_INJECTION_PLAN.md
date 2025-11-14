# Traffic Injection WebUI Implementation Plan

**Issue**: #90 - Implement traffic injection controls in webUI
**Target Release**: v2.3.0
**Created**: 2025-11-14
**Status**: Ready for implementation

---

## Overview

Add comprehensive traffic injection controls to the NIAC-Go WebUI, providing a visual interface for:
1. **Error Injection** - Inject network errors on device interfaces
2. **PCAP Replay** - Start/stop packet replay with controls

---

## Current State

### Backend APIs ✅ (Already Implemented)

#### Error Injection API
- **GET** `/api/v1/errors` - List error types and active errors
- **POST/PUT** `/api/v1/errors` - Inject/update error
  ```json
  {
    "device_ip": "192.168.1.1",
    "interface": "eth0",
    "error_type": "FCS Errors",
    "value": 50
  }
  ```
- **DELETE** `/api/v1/errors?device_ip=X&interface=Y` - Clear specific error
- **DELETE** `/api/v1/errors` - Clear all errors

**Error Types** (0-100 range):
1. FCS Errors - Frame Check Sequence errors
2. Packet Discards - Dropped packets
3. Interface Errors - Generic interface errors
4. High Utilization - Bandwidth saturation
5. High CPU - Device CPU load
6. High Memory - Device memory usage
7. High Disk - Device disk usage

#### PCAP Replay API
- **GET** `/api/v1/replay` - Get replay status
- **POST** `/api/v1/replay` - Start replay
  ```json
  {
    "file": "/path/to/capture.pcap",
    "loop_ms": 10000,
    "scale": 1.0,
    "data": "BASE64_ENCODED_PCAP"
  }
  ```
- **DELETE** `/api/v1/replay` - Stop replay

#### Supporting APIs
- **GET** `/api/v1/devices` - List all devices
- **GET** `/api/v1/files?kind=pcaps` - List available PCAP files
- **GET** `/api/v1/interfaces` - List network interfaces

### WebUI API Client 🔨 (Partially Implemented)

**Existing Functions**:
- `fetchErrorTypes()` - GET error types ✅
- `fetchReplayStatus()` - GET replay status ✅
- `startReplay(payload)` - POST replay ✅
- `stopReplay()` - DELETE replay ✅
- `fetchDevices()` - GET devices ✅
- `fetchFiles(kind)` - GET files ✅
- `fetchInterfaces()` - GET interfaces ✅

**Missing Functions**:
- `injectError(payload)` - POST/PUT error injection ❌
- `clearError(deviceIP, interface)` - DELETE specific error ❌
- `clearAllErrors()` - DELETE all errors ❌

---

## Implementation Plan

### Phase 1: API Client Extensions (30 min)

**File**: `webui/src/api/client.ts`

Add missing error injection functions:

```typescript
// Inject or update an error
export const injectError = (payload: {
  device_ip: string;
  interface: string;
  error_type: string;
  value: number;
}) =>
  request<{ success: boolean; message: string }>('/api/v1/errors', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

// Clear specific error
export const clearError = (deviceIP: string, iface: string) =>
  request<{ success: boolean; message: string }>(
    `/api/v1/errors?device_ip=${encodeURIComponent(deviceIP)}&interface=${encodeURIComponent(iface)}`,
    { method: 'DELETE' }
  );

// Clear all errors
export const clearAllErrors = () =>
  request<{ success: boolean; message: string }>('/api/v1/errors', {
    method: 'DELETE',
  });
```

**File**: `webui/src/api/types.ts`

Add type if needed:

```typescript
export interface ErrorInjectionRequest {
  device_ip: string;
  interface: string;
  error_type: string;
  value: number;
}
```

---

### Phase 2: Traffic Injection Page Component (1 hour)

**File**: `webui/src/App.tsx`

Add new page to routing:

```typescript
import { Zap } from 'lucide-react'; // Lightning icon for traffic

const pages: PageConfig[] = [
  // ... existing pages
  {
    path: '/traffic',
    label: 'Traffic Injection',
    title: 'Traffic & Error Injection',
    description: 'Inject network errors and replay PCAP traffic for testing and simulation.',
    icon: Zap,
    Component: TrafficInjectionPage,
  },
];
```

**File**: `webui/src/pages/TrafficInjectionPage.tsx` (new file)

Create main page structure:

```typescript
import { FC, useState } from 'react';
import { Card, CardContent, H2, P } from '@krisarmstrong/web-foundation';
import { ErrorInjectionPanel } from '../components/ErrorInjectionPanel';
import { ReplayControlPanel } from '../components/ReplayControlPanel';

export const TrafficInjectionPage: FC = () => {
  return (
    <div className="space-y-8">
      {/* Error Injection Section */}
      <div className="space-y-4">
        <div>
          <H2>Error Injection</H2>
          <P className="text-gray-400">
            Inject network errors on device interfaces for testing and simulation.
          </P>
        </div>
        <ErrorInjectionPanel />
      </div>

      {/* PCAP Replay Section */}
      <div className="space-y-4">
        <div>
          <H2>PCAP Replay</H2>
          <P className="text-gray-400">
            Replay captured packet traffic with loop and timing controls.
          </P>
        </div>
        <ReplayControlPanel />
      </div>
    </div>
  );
};
```

---

### Phase 3: Error Injection UI Component (1.5 hours)

**File**: `webui/src/components/ErrorInjectionPanel.tsx` (new file)

Build comprehensive error injection interface:

```typescript
import { FC, useState } from 'react';
import {
  Card,
  CardContent,
  Button,
  Select,
  Input,
  Tag,
  SmallText,
} from '@krisarmstrong/web-foundation';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchDevices,
  fetchErrorTypes,
  injectError,
  clearError,
  clearAllErrors,
} from '../api/client';

export const ErrorInjectionPanel: FC = () => {
  const { data: devices } = useApiResource(fetchDevices, []);
  const { data: errorInfo, refetch: refetchErrors } = useApiResource(fetchErrorTypes, []);

  const [selectedDevice, setSelectedDevice] = useState('');
  const [selectedInterface, setSelectedInterface] = useState('');
  const [selectedErrorType, setSelectedErrorType] = useState('');
  const [errorValue, setErrorValue] = useState(50);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleInject = async () => {
    if (!selectedDevice || !selectedInterface || !selectedErrorType) {
      setMessage({ type: 'error', text: 'Please select device, interface, and error type' });
      return;
    }

    setIsSubmitting(true);
    setMessage(null);

    try {
      await injectError({
        device_ip: selectedDevice,
        interface: selectedInterface,
        error_type: selectedErrorType,
        value: errorValue,
      });
      setMessage({ type: 'success', text: 'Error injected successfully' });
      refetchErrors();
    } catch (err) {
      setMessage({ type: 'error', text: err.message || 'Failed to inject error' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClearAll = async () => {
    if (!confirm('Clear all injected errors?')) return;

    setIsSubmitting(true);
    try {
      await clearAllErrors();
      setMessage({ type: 'success', text: 'All errors cleared' });
      refetchErrors();
    } catch (err) {
      setMessage({ type: 'error', text: err.message || 'Failed to clear errors' });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Injection Form */}
      <Card>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {/* Device Selector */}
            <div>
              <label className="block text-sm font-medium mb-2">Device</label>
              <Select
                value={selectedDevice}
                onChange={(e) => setSelectedDevice(e.target.value)}
                className="w-full"
              >
                <option value="">Select device...</option>
                {devices?.map((dev) => (
                  <option key={dev.name} value={dev.ip_addresses?.[0]}>
                    {dev.name} ({dev.ip_addresses?.[0]})
                  </option>
                ))}
              </Select>
            </div>

            {/* Interface Input */}
            <div>
              <label className="block text-sm font-medium mb-2">Interface</label>
              <Input
                type="text"
                value={selectedInterface}
                onChange={(e) => setSelectedInterface(e.target.value)}
                placeholder="e.g., eth0, GigabitEthernet0/1"
                className="w-full"
              />
            </div>

            {/* Error Type Selector */}
            <div>
              <label className="block text-sm font-medium mb-2">Error Type</label>
              <Select
                value={selectedErrorType}
                onChange={(e) => setSelectedErrorType(e.target.value)}
                className="w-full"
              >
                <option value="">Select error type...</option>
                {errorInfo?.available_types?.map((type) => (
                  <option key={type.type} value={type.type}>
                    {type.type} - {type.description}
                  </option>
                ))}
              </Select>
            </div>

            {/* Value Slider */}
            <div>
              <label className="block text-sm font-medium mb-2">
                Value: {errorValue}%
              </label>
              <input
                type="range"
                min="0"
                max="100"
                value={errorValue}
                onChange={(e) => setErrorValue(parseInt(e.target.value))}
                className="w-full"
              />
              <SmallText className="text-gray-400">
                0 = No errors, 100 = Maximum errors
              </SmallText>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div className={`p-3 rounded ${
              message.type === 'success'
                ? 'bg-green-500/10 text-green-400'
                : 'bg-red-500/10 text-red-400'
            }`}>
              {message.text}
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={handleInject}
              disabled={isSubmitting}
              variant="primary"
            >
              {isSubmitting ? 'Injecting...' : 'Inject Error'}
            </Button>
            <Button
              onClick={handleClearAll}
              disabled={isSubmitting}
              variant="secondary"
            >
              Clear All Errors
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Active Errors Table */}
      {errorInfo?.active_errors && Object.keys(errorInfo.active_errors).length > 0 && (
        <Card>
          <CardContent>
            <h3 className="text-lg font-semibold mb-4">Active Errors</h3>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-700">
                    <th className="text-left py-2">Device IP</th>
                    <th className="text-left py-2">Interface</th>
                    <th className="text-left py-2">Error Type</th>
                    <th className="text-left py-2">Value</th>
                    <th className="text-left py-2">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(errorInfo.active_errors).map(([key, errors]) =>
                    Object.entries(errors as Record<string, Record<string, number>>).map(([iface, errorTypes]) =>
                      Object.entries(errorTypes).map(([errorType, value]) => {
                        const deviceIP = key;
                        return (
                          <tr key={`${deviceIP}-${iface}-${errorType}`} className="border-b border-gray-800">
                            <td className="py-2">{deviceIP}</td>
                            <td className="py-2">{iface}</td>
                            <td className="py-2">{errorType}</td>
                            <td className="py-2">
                              <Tag variant="warning">{value}%</Tag>
                            </td>
                            <td className="py-2">
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={async () => {
                                  await clearError(deviceIP, iface);
                                  refetchErrors();
                                }}
                              >
                                Clear
                              </Button>
                            </td>
                          </tr>
                        );
                      })
                    )
                  )}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};
```

---

### Phase 4: PCAP Replay UI Component (1 hour)

**File**: `webui/src/components/ReplayControlPanel.tsx` (new file)

Build PCAP replay controls:

```typescript
import { FC, useState } from 'react';
import {
  Card,
  CardContent,
  Button,
  Select,
  Input,
  Tag,
  SmallText,
} from '@krisarmstrong/web-foundation';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchFiles,
  fetchReplayStatus,
  startReplay,
  stopReplay,
} from '../api/client';

export const ReplayControlPanel: FC = () => {
  const { data: pcapFiles } = useApiResource(() => fetchFiles('pcaps'), []);
  const { data: replayStatus, refetch: refetchStatus } = useApiResource(
    fetchReplayStatus,
    [],
    { intervalMs: 2000 }
  );

  const [selectedFile, setSelectedFile] = useState('');
  const [loopMs, setLoopMs] = useState(0);
  const [scale, setScale] = useState(1.0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleStart = async () => {
    if (!selectedFile) {
      setMessage({ type: 'error', text: 'Please select a PCAP file' });
      return;
    }

    setIsSubmitting(true);
    setMessage(null);

    try {
      await startReplay({
        file: selectedFile,
        loop_ms: loopMs,
        scale: scale,
      });
      setMessage({ type: 'success', text: 'Replay started successfully' });
      refetchStatus();
    } catch (err) {
      setMessage({ type: 'error', text: err.message || 'Failed to start replay' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleStop = async () => {
    setIsSubmitting(true);
    try {
      await stopReplay();
      setMessage({ type: 'success', text: 'Replay stopped' });
      refetchStatus();
    } catch (err) {
      setMessage({ type: 'error', text: err.message || 'Failed to stop replay' });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Status Card */}
      {replayStatus?.running && (
        <Card>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <Tag variant="success">Running</Tag>
                  <span className="font-medium">{replayStatus.file}</span>
                </div>
                <SmallText className="text-gray-400">
                  Started: {new Date(replayStatus.started_at).toLocaleString()}
                  {replayStatus.loop_ms > 0 && ` • Looping every ${replayStatus.loop_ms}ms`}
                  {replayStatus.scale !== 1.0 && ` • Scale: ${replayStatus.scale}x`}
                </SmallText>
              </div>
              <Button
                onClick={handleStop}
                disabled={isSubmitting}
                variant="danger"
              >
                Stop Replay
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Control Card */}
      <Card>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {/* File Selector */}
            <div className="col-span-2">
              <label className="block text-sm font-medium mb-2">PCAP File</label>
              <Select
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                className="w-full"
                disabled={replayStatus?.running}
              >
                <option value="">Select PCAP file...</option>
                {pcapFiles?.map((file) => (
                  <option key={file.path} value={file.path}>
                    {file.filename} ({(file.size_bytes / 1024).toFixed(1)} KB)
                  </option>
                ))}
              </Select>
            </div>

            {/* Loop Interval */}
            <div>
              <label className="block text-sm font-medium mb-2">
                Loop Interval (ms)
              </label>
              <Input
                type="number"
                min="0"
                step="1000"
                value={loopMs}
                onChange={(e) => setLoopMs(parseInt(e.target.value) || 0)}
                placeholder="0 = no loop"
                disabled={replayStatus?.running}
              />
              <SmallText className="text-gray-400">
                0 = Play once, &gt;0 = Loop with delay
              </SmallText>
            </div>

            {/* Time Scale */}
            <div>
              <label className="block text-sm font-medium mb-2">
                Time Scale
              </label>
              <Input
                type="number"
                min="0.1"
                max="10"
                step="0.1"
                value={scale}
                onChange={(e) => setScale(parseFloat(e.target.value) || 1.0)}
                disabled={replayStatus?.running}
              />
              <SmallText className="text-gray-400">
                1.0 = Original timing, 2.0 = 2x faster, 0.5 = 2x slower
              </SmallText>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div className={`p-3 rounded ${
              message.type === 'success'
                ? 'bg-green-500/10 text-green-400'
                : 'bg-red-500/10 text-red-400'
            }`}>
              {message.text}
            </div>
          )}

          {/* Action Button */}
          {!replayStatus?.running && (
            <Button
              onClick={handleStart}
              disabled={isSubmitting}
              variant="primary"
              className="w-full"
            >
              {isSubmitting ? 'Starting...' : 'Start Replay'}
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
```

---

### Phase 5: Build & Deploy (15 min)

**Commands**:
```bash
cd webui
npm install  # If dependencies changed
npm run build
```

The build output goes to `pkg/api/ui/assets/` and is embedded in the Go binary.

---

### Phase 6: Documentation (30 min)

**File**: `docs/REST_API.md`

Update with traffic injection examples:

```markdown
### Traffic Injection

The WebUI provides visual controls for traffic injection at `/traffic`.

**Error Injection**:
- Select device and interface
- Choose error type (FCS, Discards, Utilization, CPU, Memory, Disk)
- Set error value (0-100%)
- View active errors table
- Clear individual or all errors

**PCAP Replay**:
- Select PCAP file from available files
- Set loop interval (0 = play once)
- Set time scale (1.0 = original timing)
- View replay status
- Start/stop controls
```

**File**: `docs/TRAFFIC_INJECTION.md` (new)

Create comprehensive user guide with screenshots and examples.

---

## Testing Plan

### Manual Testing

1. **Error Injection**:
   - [ ] Select device and interface
   - [ ] Inject FCS error at 50%
   - [ ] Verify error appears in active errors table
   - [ ] Verify error appears in SNMP query
   - [ ] Clear specific error
   - [ ] Inject multiple errors on different devices
   - [ ] Clear all errors
   - [ ] Verify error validation (0-100 range)

2. **PCAP Replay**:
   - [ ] Select PCAP file from dropdown
   - [ ] Start replay without loop
   - [ ] Verify replay status shows running
   - [ ] Stop replay
   - [ ] Start replay with 5000ms loop
   - [ ] Verify looping behavior
   - [ ] Test time scale (0.5x, 2.0x)
   - [ ] Verify replay stops on error

3. **Integration**:
   - [ ] Verify WebUI navigation works
   - [ ] Test with API token authentication
   - [ ] Test error handling (no simulation running)
   - [ ] Test concurrent operations

---

## Success Criteria

- [ ] WebUI has "Traffic Injection" page in navigation
- [ ] Error injection form works with all 7 error types
- [ ] Active errors table displays correctly
- [ ] Clear operations work (individual and all)
- [ ] PCAP replay controls work (start/stop/loop/scale)
- [ ] Replay status updates in real-time
- [ ] Error messages display properly
- [ ] All builds pass
- [ ] Documentation updated
- [ ] Issue #90 closed with demo screenshots

---

## Files to Create/Modify

### New Files (4):
- `docs/TRAFFIC_INJECTION_PLAN.md` ✅
- `docs/TRAFFIC_INJECTION.md`
- `webui/src/pages/TrafficInjectionPage.tsx`
- `webui/src/components/ErrorInjectionPanel.tsx`
- `webui/src/components/ReplayControlPanel.tsx`

### Modified Files (4):
- `webui/src/App.tsx` - Add routing
- `webui/src/api/client.ts` - Add API functions
- `webui/src/api/types.ts` - Add types (if needed)
- `docs/REST_API.md` - Update documentation

---

## Estimated Time

- Phase 1: API Client Extensions - **30 min**
- Phase 2: Page Component - **1 hour**
- Phase 3: Error Injection UI - **1.5 hours**
- Phase 4: Replay UI - **1 hour**
- Phase 5: Build & Deploy - **15 min**
- Phase 6: Documentation - **30 min**

**Total: ~4.5 hours**

---

**End of Plan**
