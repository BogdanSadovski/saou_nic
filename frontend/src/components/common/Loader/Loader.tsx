import React from 'react';
import styles from './Loader.module.css';

export type LoaderSize = 'sm' | 'md' | 'lg';
export type LoaderVariant = 'spinner' | 'dots' | 'bars';

export interface LoaderProps {
  size?: LoaderSize;
  variant?: LoaderVariant;
  label?: string;
  fullWidth?: boolean;
  className?: string;
}

const Loader: React.FC<LoaderProps> = ({
  size = 'md',
  variant = 'spinner',
  label,
  fullWidth = false,
  className = '',
}) => {
  const classes = [
    styles.loader,
    styles[size],
    fullWidth ? styles.fullWidth : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  const renderLoader = () => {
    switch (variant) {
      case 'dots':
        return (
          <div className={styles.dots}>
            <span />
            <span />
            <span />
          </div>
        );
      case 'bars':
        return (
          <div className={styles.bars}>
            <span />
            <span />
            <span />
            <span />
          </div>
        );
      case 'spinner':
      default:
        return <div className={styles.spinner} />;
    }
  };

  return (
    <div className={classes} role="status" aria-label={label || 'Loading'}>
      {renderLoader()}
      {label && <span className={styles.label}>{label}</span>}
      <span className="sr-only">{label || 'Loading'}</span>
    </div>
  );
};

export default Loader;
