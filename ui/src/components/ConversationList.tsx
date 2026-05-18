import { type FC, memo, useCallback, useMemo, useState } from 'react';
import type { PcapPacket } from '../api/types';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import {
  type Conversation,
  extractConversations,
  getConversationDuration,
} from '../utils/conversations';
import { formatBytes, formatDurationSeconds } from '../utils/format';
import { getProtocolColor } from '../utils/protocol-colors';
import type { Packet } from './PacketList';

type SortField = 'packets' | 'bytes' | 'duration' | 'protocol';
type SortDirection = 'asc' | 'desc';

interface ConversationListProps {
  packets: (Packet | PcapPacket)[];
  onSelectConversation: (filterExpression: string) => void;
}

/**
 * Conversation List Component
 *
 * Displays a table of TCP/UDP conversations extracted from the packet list.
 * Clicking a row sets the display filter to show only that conversation's packets.
 */
export const ConversationList: FC<ConversationListProps> = memo(
  ({ packets, onSelectConversation }) => {
    const [sortField, setSortField] = useState<SortField>('packets');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

    const conversations = useMemo(() => extractConversations(packets), [packets]);

    const sorted = useMemo(() => {
      const copy = [...conversations];
      copy.sort((a, b) => {
        let cmp = 0;
        switch (sortField) {
          case 'packets':
            cmp = a.packets - b.packets;
            break;
          case 'bytes':
            cmp = a.bytes - b.bytes;
            break;
          case 'duration':
            cmp = getConversationDuration(a) - getConversationDuration(b);
            break;
          case 'protocol':
            cmp = a.protocol.localeCompare(b.protocol);
            break;
        }
        return sortDirection === 'desc' ? -cmp : cmp;
      });
      return copy;
    }, [conversations, sortField, sortDirection]);

    const handleSort = useCallback(
      (field: SortField) => {
        if (sortField === field) {
          setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
        } else {
          setSortField(field);
          setSortDirection('desc');
        }
      },
      [sortField],
    );

    const sortIndicator = useCallback(
      (field: SortField) => {
        if (sortField !== field) return '';
        return sortDirection === 'desc' ? ' ▼' : ' ▲';
      },
      [sortField, sortDirection],
    );

    if (conversations.length === 0) {
      return (
        <Card className="border-white/5 bg-bg-surface/70">
          <CardContent>
            <div className="text-center py-8 text-text-muted">
              <p className="text-sm">No TCP/UDP conversations found</p>
              <SmallText>Conversations require packets with port information</SmallText>
            </div>
          </CardContent>
        </Card>
      );
    }

    return (
      <Card className="border-white/5 bg-bg-surface/70">
        <CardContent>
          <div className="mb-3 flex items-center justify-between">
            <SmallText className="text-text-muted">
              {conversations.length} conversation{conversations.length !== 1 ? 's' : ''}
            </SmallText>
            <SmallText className="text-text-muted">Click a row to filter packets</SmallText>
          </div>

          <div className="overflow-auto rounded-lg border border-white/5 max-h-[500px]">
            <table className="min-w-full divide-y divide-white/5">
              <thead className="bg-bg-surface/80 sticky top-0 z-10">
                <tr>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                    Endpoint A
                  </th>
                  <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                    Endpoint B
                  </th>
                  <th
                    className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted cursor-pointer hover:text-brand-300 select-none"
                    onClick={() => handleSort('protocol')}
                  >
                    Protocol{sortIndicator('protocol')}
                  </th>
                  <th
                    className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-text-muted cursor-pointer hover:text-brand-300 select-none"
                    onClick={() => handleSort('packets')}
                  >
                    Packets{sortIndicator('packets')}
                  </th>
                  <th
                    className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-text-muted cursor-pointer hover:text-brand-300 select-none"
                    onClick={() => handleSort('bytes')}
                  >
                    Bytes{sortIndicator('bytes')}
                  </th>
                  <th
                    className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-text-muted cursor-pointer hover:text-brand-300 select-none"
                    onClick={() => handleSort('duration')}
                  >
                    Duration{sortIndicator('duration')}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {sorted.map((conv) => (
                  <ConversationRow
                    key={conv.id}
                    conversation={conv}
                    onClick={() => onSelectConversation(conv.filterExpression)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    );
  },
);

ConversationList.displayName = 'ConversationList';

const ConversationRow: FC<{
  conversation: Conversation;
  onClick: () => void;
}> = memo(({ conversation, onClick }) => {
  const duration = getConversationDuration(conversation);

  return (
    <tr onClick={onClick} className="cursor-pointer hover:bg-bg-surface/50 transition-colors">
      <td className="px-3 py-2 text-text-primary text-sm font-mono">{conversation.endpointA}</td>
      <td className="px-3 py-2 text-text-primary text-sm font-mono">{conversation.endpointB}</td>
      <td className="px-3 py-2">
        <Tag colorScheme={getProtocolColor(conversation.protocol)} className="text-xs">
          {conversation.protocol}
        </Tag>
      </td>
      <td className="px-3 py-2 text-text-secondary text-sm text-right font-mono">
        {conversation.packets}
      </td>
      <td className="px-3 py-2 text-text-secondary text-sm text-right font-mono">
        {formatBytes(conversation.bytes)}
      </td>
      <td className="px-3 py-2 text-text-muted text-sm text-right font-mono">
        {formatDurationSeconds(duration)}
      </td>
    </tr>
  );
});

ConversationRow.displayName = 'ConversationRow';

export default ConversationList;
