import type { CSSProperties } from "react";

type Props = {
  size?: number;
  className?: string;
  style?: CSSProperties;
};

/**
 * Графический знак белорусского рубля (утверждён НБ РБ, 2026).
 *
 * Стилизованная лигатура «Br» с единым горизонтальным штрихом,
 * пересекающим обе буквы. У Unicode-точки для этого знака пока
 * нет, поэтому используем встроенный SVG-глиф. Цвет наследуется
 * через `currentColor`, размер задаётся в пикселях.
 */
export function BynSign({ size = 14, className, style }: Props) {
  const w = size * 1.5;
  return (
    <svg
      role="img"
      aria-label="белорусский рубль"
      width={w}
      height={size}
      viewBox="0 0 30 20"
      className={className}
      style={{ display: "inline-block", verticalAlign: "-0.1em", ...style }}
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Буква B */}
      <path
        d="M2 2 L2 18 L9.5 18 C12 18 14 16.3 14 13.7 C14 12 13 10.6 11.5 10.1 C12.6 9.5 13.4 8.3 13.4 6.8 C13.4 4.4 11.6 2.7 9.2 2.7 L2 2.7 Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      {/* Внутренние перегородки B */}
      <line x1="4.4" y1="2.7" x2="4.4" y2="18" stroke="currentColor" strokeWidth="1.6" />
      <line x1="4.4" y1="10.1" x2="11.5" y2="10.1" stroke="currentColor" strokeWidth="1.4" />
      {/* Буква r */}
      <path
        d="M17 7 L17 18 M17 11 C17 8.7 18.6 7 21 7 L22.5 7"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Горизонтальный штрих, пересекающий обе буквы */}
      <line
        x1="0.5"
        y1="14.6"
        x2="25"
        y2="14.6"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
    </svg>
  );
}
