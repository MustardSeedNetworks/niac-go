import type { CompiledAttachment, SimulationStatus } from '../../api/types';

/**
 * The port pool the session is bound to, or undefined when its attachment
 * names a whole network: only a pool has ports to move a client between.
 */
export const sessionPool = (fabric: SimulationStatus['fabric']): CompiledAttachment | undefined =>
  fabric?.topology.attachments?.find(
    (attachment) => attachment.name === fabric.topology.binding.attachment,
  );
