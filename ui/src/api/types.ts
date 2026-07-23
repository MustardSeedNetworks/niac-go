/**
 * API Types - Re-exports from focused modules
 *
 * This file provides backwards compatibility by re-exporting all types
 * from their respective modules. New code should import directly from
 * the specific module files:
 *
 * - api-response-types.ts: General API responses (stack stats, history, etc.)
 * - device-config-types.ts: Device configuration (SNMP, LLDP, DHCP, etc.)
 * - debug-types.ts: Debug console and logging types
 * - template-types.ts: Configuration template types
 * - pcap-types.ts: PCAP analyzer types
 * - walk-analyze-types.ts: Walk analyzer types
 */

// API Response Types
export type {
  AlertConfig,
  ConfigDocument,
  ConfigUpdateRequest,
  DeviceSummary,
  ErrorInjectionInfo,
  ErrorType,
  FileEntry,
  HistoryRecord,
  InterfacesResponse,
  ModelDescriptor,
  NeighborRecord,
  NetworkInterface,
  ReplayRequest,
  ReplayState,
  RuntimeStatus,
  SegmentSummary,
  SimulationRequest,
  SimulationStatus,
  StackStatsResponse,
  StandaloneCaptureRequest,
  StandaloneCaptureStatus,
  SynthesizeWalkRequest,
  SynthesizeWalkResponse,
  TopologyGraph,
  TopologyLink,
  TopologyNode,
  VersionInfo,
  WalkBatchValidationResponse,
  WalkValidationIssue,
  WalkValidationResponse,
  WalkValidationResult,
} from './api-response-types';
// Debug Types
export type {
  DebugLevel,
  DebugLevelResponse,
  LogEntry,
  LogLevel,
  Protocol,
  UpdateDebugLevelRequest,
} from './debug-types';
// Device Configuration Types
export type {
  AddMib,
  ARPAnnouncementConfig,
  CDPConfig,
  ChassisIDType,
  CloneDeviceRequest,
  ConfigSchema,
  CreateDeviceRequest,
  Device,
  DeviceBatchDeleteRequest,
  DeviceBatchDeleteResponse,
  DeviceBatchDeleteResult,
  DeviceDetailResponse,
  DeviceInterface,
  DeviceListResponse,
  DeviceMutationResponse,
  DeviceType,
  DHCPConfig,
  DHCPLease,
  DHCPv6Config,
  DHCPv6Pool,
  DNSConfig,
  DNSRecord,
  EDPConfig,
  FDPConfig,
  FdbTableConfig,
  FTPConfig,
  FTPUser,
  HTTPConfig,
  HTTPEndpoint,
  ICMPConfig,
  ICMPRouter,
  ICMPRouterAdvertisement,
  ICMPv6Config,
  ICMPv6PrefixInfo,
  ICMPv6RouterAdvertisement,
  IPerf3Config,
  JSONSchemaProperty,
  LinkStateTrapConfig,
  LLDPConfig,
  MibType,
  NetBIOSConfig,
  NetBIOSName,
  NetBIOSNodeType,
  NetBIOSService,
  OSFingerprintConfig,
  OSType,
  PeriodicPingConfig,
  RandomTrafficConfig,
  RandomTrafficPattern,
  SNMPAgent,
  SSHConfig,
  STPConfig,
  STPVersion,
  SyslogConfig,
  TrafficConfig,
  TrapConfig,
  TrapTriggerConfig,
  TTLConfig,
  UpdateDeviceRequest,
} from './device-config-types';
export type {
  FabricBinding,
  FabricDhcpScope,
  FabricDiagnostic,
  FabricInterface,
  FabricNetwork,
  FabricRoute,
  SimulationPreflightReport,
  SimulationPreflightRequest,
} from './fabric-types';

// PCAP Types
export type {
  CaptureFilterResponse,
  PcapAnalysisResult,
  PcapPacket,
  PcapStats,
  PcapUploadRequest,
  PcapUploadResponse,
} from './pcap-types';

// Template Types
export type {
  LibraryNetwork,
  LibraryNetworkContent,
  Template,
  TemplateContent,
  UploadLibraryNetworkRequest,
  UploadLibraryNetworkResponse,
  UploadTemplateRequest,
  UploadTemplateResponse,
  UseTemplateRequest,
  UseTemplateResponse,
} from './template-types';

// Walk Analyzer Types
export type {
  WalkAnalysis,
  WalkAnalyzeDevice,
  WalkAnalyzeInterface,
  WalkAnalyzeNeighbor,
  WalkAnalyzeResponse,
  WalkAnalyzeStats,
} from './walk-analyze-types';
