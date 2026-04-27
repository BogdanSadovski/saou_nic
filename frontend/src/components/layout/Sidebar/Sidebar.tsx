import React from 'react';

interface NavItem {
  id: string;
  label: string;
  icon?: React.ReactNode;
  path: string;
}

interface SidebarProps {
  items?: NavItem[];
  activeItem?: string;
  onNavigate?: (path: string) => void;
  collapsed?: boolean;
}

const defaultNavItems: NavItem[] = [
  { id: 'dashboard', label: 'Dashboard', path: '/dashboard' },
  { id: 'resume', label: 'Resume', path: '/resume' },
  { id: 'interviews', label: 'Interviews', path: '/interviews' },
  { id: 'reports', label: 'Reports', path: '/reports' },
  { id: 'github', label: 'GitHub', path: '/github' },
];

const Sidebar: React.FC<SidebarProps> = ({
  items = defaultNavItems,
  activeItem,
  onNavigate,
  collapsed = false,
}) => {
  return (
    <aside className={`sidebar ${collapsed ? 'sidebar--collapsed' : ''}`}>
      <nav className="sidebar__nav">
        <ul className="sidebar__list">
          {items.map((item) => (
            <li key={item.id} className="sidebar__item">
              <button
                className={`sidebar__link ${activeItem === item.id ? 'sidebar__link--active' : ''}`}
                onClick={() => onNavigate?.(item.path)}
              >
                {item.icon && <span className="sidebar__icon">{item.icon}</span>}
                {!collapsed && <span className="sidebar__label">{item.label}</span>}
              </button>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
};

export default Sidebar;
