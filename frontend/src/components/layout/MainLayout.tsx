import React, { useState } from 'react';
import { Header } from './Header';
import { Sidebar } from './Sidebar';
import { Footer } from './Footer';

interface MainLayoutProps {
  children: React.ReactNode;
  pageTitle?: string;
}

const MainLayout: React.FC<MainLayoutProps> = ({ children, pageTitle }) => {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const toggleSidebar = () => {
    setSidebarCollapsed((prev) => !prev);
  };

  return (
    <div className="main-layout">
      <Sidebar collapsed={sidebarCollapsed} />
      <div className="main-layout__content">
        <Header title={pageTitle} onMenuToggle={toggleSidebar} />
        <main className="main-layout__main">{children}</main>
        <Footer />
      </div>
    </div>
  );
};

export default MainLayout;
