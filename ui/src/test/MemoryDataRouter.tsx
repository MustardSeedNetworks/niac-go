import { createContext, type ReactNode, useContext, useState } from 'react';
import { createMemoryRouter, RouterProvider } from 'react-router';

const Content = createContext<ReactNode>(null);
const RouteContent = () => useContext(Content);

export function MemoryDataRouter({
  children,
  initialEntries = ['/'],
}: {
  children: ReactNode;
  initialEntries?: string[];
}) {
  const [router] = useState(() =>
    createMemoryRouter([{ path: '*', element: <RouteContent /> }], { initialEntries }),
  );
  return (
    <Content value={children}>
      <RouterProvider router={router} />
    </Content>
  );
}
