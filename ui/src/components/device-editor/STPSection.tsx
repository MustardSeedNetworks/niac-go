import type { FC } from "react";
import type { STPConfig } from "../../api/types";
import { CollapsibleSection, FormField } from "../form";
import type { ProtocolSectionProps } from "./types";
import { inputClassName } from "./types";

export const STPSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateStp = (config: STPConfig | undefined) => {
    onUpdate("stp", config);
  };

  return (
    <CollapsibleSection
      title="STP (Spanning Tree Protocol)"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.stp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateStp(enabled ? ({ enabled: true, bridge_priority: 32768 } as STPConfig) : undefined);
      }}
    >
      {device.stp?.enabled && (
        <div className="grid gap-4 md:grid-cols-2">
          <FormField
            label="Bridge Priority"
            helpText="STP bridge priority (0-61440, in steps of 4096)"
          >
            <input
              type="number"
              value={device.stp.bridge_priority ?? 32768}
              onChange={(e) =>
                updateStp({ ...device.stp!, bridge_priority: parseInt(e.target.value, 10) })
              }
              min={0}
              max={61440}
              step={4096}
              className={inputClassName}
            />
          </FormField>
        </div>
      )}
    </CollapsibleSection>
  );
};
