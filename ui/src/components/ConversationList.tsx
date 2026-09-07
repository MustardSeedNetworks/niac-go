import { type FC, memo, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../api/types';
import { Card, CardContent } from '../ui/Card';
import { DataTable, type DataTableColumn } from '../ui/DataTable';
import { InfoPopover } from '../ui/InfoPopover';
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
    const { t } = useTranslation('common');
    const { t: tHelp } = useTranslation('help');
    const { t: tPages } = useTranslation('pages');
    const conversations = useMemo(() => extractConversations(packets), [packets]);

    const columns: DataTableColumn<Conversation>[] = [
      {
        key: 'endpointA',
        header: tPages('packets.conversations.headerEndpointA'),
        cellClassName: 'text-text-primary text-sm font-mono',
        cell: (conv) => conv.endpointA,
      },
      {
        key: 'endpointB',
        header: tPages('packets.conversations.headerEndpointB'),
        cellClassName: 'text-text-primary text-sm font-mono',
        cell: (conv) => conv.endpointB,
      },
      {
        key: 'protocol',
        header: tPages('packets.list.headerProtocol'),
        sortAccessor: (conv) => conv.protocol,
        cell: (conv) => (
          <Tag colorScheme={getProtocolColor(conv.protocol)} className="text-xs">
            {conv.protocol}
          </Tag>
        ),
      },
      {
        key: 'packets',
        header: tPages('packets.conversations.headerPackets'),
        align: 'right',
        sortAccessor: (conv) => conv.packets,
        cellClassName: 'text-text-secondary text-sm font-mono',
        cell: (conv) => conv.packets,
      },
      {
        key: 'bytes',
        header: tPages('packets.conversations.headerBytes'),
        align: 'right',
        sortAccessor: (conv) => conv.bytes,
        cellClassName: 'text-text-secondary text-sm font-mono',
        cell: (conv) => formatBytes(conv.bytes),
      },
      {
        key: 'duration',
        header: tPages('packets.stats.duration'),
        align: 'right',
        sortAccessor: (conv) => getConversationDuration(conv),
        cellClassName: 'text-text-muted text-sm font-mono',
        cell: (conv) => formatDurationSeconds(getConversationDuration(conv)),
      },
    ];

    if (conversations.length === 0) {
      return (
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent>
            <div className="text-center py-8 text-text-muted">
              <p className="text-sm">{tPages('packets.conversations.emptyTitle')}</p>
              <SmallText>{tPages('packets.conversations.emptyDescription')}</SmallText>
            </div>
          </CardContent>
        </Card>
      );
    }

    return (
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent>
          <div className="mb-heading flex-between">
            <SmallText className="text-text-muted flex items-center gap-1">
              {t('plurals.conversationCount', { count: conversations.length })}
              <span>{t('jargon.groupedByTerm', { term: '5-tuple' })}</span>
              <InfoPopover label={t('jargon.ariaLabel', { term: '5-tuple' })} title="5-tuple">
                {tHelp('jargon.fiveTuple')}
              </InfoPopover>
            </SmallText>
            <SmallText className="text-text-muted">
              {tPages('packets.conversations.clickToFilterHint')}
            </SmallText>
          </div>

          <DataTable
            rows={conversations}
            columns={columns}
            getRowKey={(conv) => conv.id}
            defaultSort={{ key: 'packets', direction: 'desc' }}
            onRowClick={(conv) => onSelectConversation(conv.filterExpression)}
            rowClassName={() => 'hover:bg-bg-surface/50 transition-colors'}
            stickyHeader
            containerClassName="max-h-[500px] overflow-y-auto"
            emptyMessage={null}
          />
        </CardContent>
      </Card>
    );
  },
);

ConversationList.displayName = 'ConversationList';
