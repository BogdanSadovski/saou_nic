import type { PropsWithChildren } from "react";

import { cn } from "@/shared/lib/cn";

type ModalProps = PropsWithChildren<{
  isOpen: boolean;
  title: string;
  onClose: () => void;
}>;

export function Modal({ isOpen, title, onClose, children }: ModalProps) {
  if (!isOpen) return null;

  return (
    <div className="modal-overlay" role="presentation" onClick={onClose}>
      <div
        className={cn("modal-card", "glass-card")}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h3>{title}</h3>
          <button className="modal-close" onClick={onClose} type="button">
            x
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
