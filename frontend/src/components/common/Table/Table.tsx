import React from 'react';
import styles from './Table.module.css';

export interface Column<T = Record<string, unknown>> {
  key: string;
  header: string;
  render?: (value: unknown, row: T, index: number) => React.ReactNode;
  sortable?: boolean;
  width?: string;
  className?: string;
}

export interface TableProps<T = Record<string, unknown>> {
  columns: Column<T>[];
  data: T[];
  onRowClick?: (row: T, index: number) => void;
  loading?: boolean;
  emptyMessage?: string;
  className?: string;
  striped?: boolean;
  hoverable?: boolean;
  size?: 'sm' | 'md';
}

function Table<T extends Record<string, unknown>>({
  columns,
  data,
  onRowClick,
  loading = false,
  emptyMessage = 'No data available',
  className = '',
  striped = false,
  hoverable = false,
  size = 'md',
}: TableProps<T>) {
  const renderCell = (column: Column<T>, row: T, rowIndex: number): React.ReactNode => {
    const value = row[column.key as keyof T];

    if (column.render) {
      return column.render(value, row, rowIndex);
    }

    return value != null ? String(value) : '';
  };

  if (loading) {
    return (
      <div className={`${styles.container} ${className}`}>
        <div className={styles.loading}>Loading...</div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className={`${styles.container} ${className}`}>
        <div className={styles.empty}>{emptyMessage}</div>
      </div>
    );
  }

  return (
    <div className={`${styles.container} ${className}`}>
      <table
        className={[
          styles.table,
          styles[size],
          striped ? styles.striped : '',
          hoverable ? styles.hoverable : '',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                style={column.width ? { width: column.width } : undefined}
                className={column.className}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((row, rowIndex) => (
            <tr
              key={rowIndex}
              onClick={() => onRowClick?.(row, rowIndex)}
              className={onRowClick ? styles.clickable : ''}
            >
              {columns.map((column) => (
                <td key={column.key} className={column.className}>
                  {renderCell(column, row, rowIndex)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default Table;
