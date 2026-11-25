import { memo, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../api/types';
import { Card } from '../shared/Card';
import { CategoryChip } from '../shared/CategoryChip';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { ContainedTable } from '../ui/ContainedTable';
import { SmallText } from '../ui/Typography';
import { useTableLabels } from '../ui/tableLabels';
import {
  type Conversation,
  extractConversations,
  getConversationDuration,
} from '../utils/conversations';
import { formatBytes, formatDurationSeconds } from '../utils/format';
import { GlossaryPopover } from './GlossaryPopover';
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
export const ConversationList = memo(function ConversationList({
  packets,
  onSelectConversation,
}: ConversationListProps) {
  const { t } = useTranslation('common');
  const { t: tPages } = useTranslation('pages');
  const labels = useTableLabels<Conversation>(t('labels.conversations'));
  const conversations = useMemo(() => extractConversations(packets), [packets]);

  const columns: DataTableColumn<Conversation>[] = [
    {
      key: 'endpointA',
      header: tPages('packets.conversations.headerEndpointA'),
      sortValue: (conv) => conv.endpointA,
      cell: (conv) => <span className="font-mono text-text-primary">{conv.endpointA}</span>,
    },
    {
      key: 'endpointB',
      header: tPages('packets.conversations.headerEndpointB'),
      sortValue: (conv) => conv.endpointB,
      cell: (conv) => <span className="font-mono text-text-primary">{conv.endpointB}</span>,
    },
    {
      key: 'protocol',
      header: tPages('packets.list.headerProtocol'),
      sortValue: (conv) => conv.protocol,
      cell: (conv) => <CategoryChip label={conv.protocol} />,
    },
    {
      key: 'packets',
      header: tPages('packets.conversations.headerPackets'),
      align: 'right',
      sortValue: (conv) => conv.packets,
      cell: (conv) => <span className="font-mono">{conv.packets}</span>,
    },
    {
      key: 'bytes',
      header: tPages('packets.conversations.headerBytes'),
      align: 'right',
      sortValue: (conv) => conv.bytes,
      cell: (conv) => <span className="font-mono">{formatBytes(conv.bytes)}</span>,
    },
    {
      key: 'duration',
      header: tPages('packets.stats.duration'),
      align: 'right',
      sortValue: (conv) => getConversationDuration(conv),
      cell: (conv) => (
        <span className="font-mono text-text-muted">
          {formatDurationSeconds(getConversationDuration(conv))}
        </span>
      ),
    },
  ];

  if (conversations.length === 0) {
    return (
      <Card>
        <EmptyState
          title={tPages('packets.conversations.emptyTitle')}
          hint={tPages('packets.conversations.emptyDescription')}
        />
      </Card>
    );
  }

  return (
    <Card>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <SmallText className="flex items-center gap-1 text-text-muted">
          {t('plurals.conversationCount', { count: conversations.length })}
          <span>{t('jargon.groupedByTerm', { term: '5-tuple' })}</span>
          <GlossaryPopover term="fiveTuple" />
        </SmallText>
        <SmallText className="text-text-muted">
          {tPages('packets.conversations.clickToFilterHint')}
        </SmallText>
      </div>

      <ContainedTable label={t('labels.conversations')}>
        <DataTable
          rows={conversations}
          columns={columns}
          getRowKey={(conv) => conv.id}
          caption={t('labels.conversations')}
          labels={labels}
          defaultSort={{ key: 'packets', direction: 'desc' }}
          onRowClick={(conv) => onSelectConversation(conv.filterExpression)}
          virtualize={{ rowHeight: 44 }}
          maxHeight={500}
          empty={null}
        />
      </ContainedTable>
    </Card>
  );
});
