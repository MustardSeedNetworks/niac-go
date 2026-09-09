import { useCallback, useReducer } from 'react';
import type { Packet } from '../../components/PacketList';

export const MAX_PACKETS = 100;

interface BufferState {
  retained: Packet[];
  visible: Packet[];
  paused: boolean;
  pending: number;
  evicted: number;
}

type BufferAction = { type: 'packet'; packet: Packet } | { type: 'toggle' } | { type: 'clear' };
const emptyBuffer: BufferState = {
  retained: [],
  visible: [],
  paused: false,
  pending: 0,
  evicted: 0,
};

function reduceBuffer(state: BufferState, action: BufferAction): BufferState {
  if (action.type === 'clear') return { ...emptyBuffer, paused: state.paused };
  if (action.type === 'toggle') {
    return { ...state, paused: !state.paused, visible: state.retained, pending: 0 };
  }
  const retained = [...state.retained, action.packet].slice(-MAX_PACKETS);
  return {
    ...state,
    retained,
    visible: state.paused ? state.visible : retained,
    pending: state.paused ? state.pending + 1 : 0,
    evicted: state.evicted + Number(state.retained.length === MAX_PACKETS),
  };
}

export function usePacketBuffer() {
  const [state, dispatch] = useReducer(reduceBuffer, emptyBuffer);
  const appendPacket = useCallback((packet: Packet) => dispatch({ type: 'packet', packet }), []);
  const clearPackets = useCallback(() => dispatch({ type: 'clear' }), []);
  const togglePause = useCallback(() => dispatch({ type: 'toggle' }), []);
  return { ...state, appendPacket, clearPackets, togglePause };
}
