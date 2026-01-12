import type { FC } from "react";
import type { FileEntry, SNMPAgent } from "../../api/types";
import { CollapsibleSection, FormField } from "../form";
import type { SNMPSectionProps } from "./types";
import { inputClassName, selectClassName } from "./types";

export const SNMPSection: FC<SNMPSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
  walkFiles,
}) => {
  const updateSnmp = (config: SNMPAgent | undefined) => {
    onUpdate("snmp_agent", config);
  };

  return (
    <CollapsibleSection
      title="SNMP Agent"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={!!device.snmp_agent}
      onEnableChange={(enabled) => {
        if (enabled) {
          updateSnmp({ community: "public", sys_name: device.hostname } as SNMPAgent);
        } else {
          updateSnmp(undefined);
        }
      }}
    >
      {device.snmp_agent && (
        <div className="grid gap-4 md:grid-cols-2">
          <FormField label="Community String" helpText="SNMP v1/v2c community string">
            <input
              type="text"
              value={device.snmp_agent.community || ""}
              onChange={(e) => updateSnmp({ ...device.snmp_agent!, community: e.target.value })}
              placeholder="public"
              className={inputClassName}
            />
          </FormField>

          <FormField label="System Name" helpText="SNMP sysName value">
            <input
              type="text"
              value={device.snmp_agent.sys_name || ""}
              onChange={(e) => updateSnmp({ ...device.snmp_agent!, sys_name: e.target.value })}
              placeholder="Device hostname"
              className={inputClassName}
            />
          </FormField>

          <FormField
            label="Walk File"
            helpText="Path to SNMP walk file for emulation"
            className="md:col-span-2"
          >
            <select
              value={device.snmp_agent.walk_file || ""}
              onChange={(e) => updateSnmp({ ...device.snmp_agent!, walk_file: e.target.value })}
              className={selectClassName}
            >
              <option value="">Select a walk file...</option>
              {walkFiles?.map((file: FileEntry) => (
                <option key={file.path} value={file.path}>
                  {file.name}
                </option>
              ))}
            </select>
          </FormField>
        </div>
      )}
    </CollapsibleSection>
  );
};
