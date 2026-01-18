import type { FC, ReactNode } from 'react';

type TagColorScheme = 'gray' | 'red' | 'green' | 'blue' | 'yellow' | 'purple' | 'violet' | 'cyan';

interface TagProps {
  children: ReactNode;
  colorScheme?: TagColorScheme;
  className?: string;
}

const colorStyles: Record<TagColorScheme, string> = {
  gray: 'bg-gray-500/20 text-gray-300 border-gray-500/30',
  red: 'bg-red-500/20 text-red-300 border-red-500/30',
  green: 'bg-green-500/20 text-green-300 border-green-500/30',
  blue: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
  yellow: 'bg-yellow-500/20 text-yellow-300 border-yellow-500/30',
  purple: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
  violet: 'bg-violet-500/20 text-violet-300 border-violet-500/30',
  cyan: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30',
};

export const Tag: FC<TagProps> = ({ children, colorScheme = 'gray', className = '' }) => (
  <span
    className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${colorStyles[colorScheme]} ${className}`}
  >
    {children}
  </span>
);
