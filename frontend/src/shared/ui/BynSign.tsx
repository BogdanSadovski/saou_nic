import type { CSSProperties } from "react";

type Props = {
  size?: number;
  className?: string;
  style?: CSSProperties;
};

/**
 * Графический знак белорусского рубля (утверждён НБ РБ, 2026).
 *
 * Стилизованная лигатура «Br» с одним коротким горизонтальным штрихом
 * ПОД литерой B (как у ₽ — Cyrillic R с подчерком), а не сквозь весь
 * знак — иначе получается визуальный strikethrough и читается как
 * «зачёркнутая цена», а не как валютный знак.
 *
 * Цвет наследуется через `currentColor`, размер задаётся в пикселях.
 */
export function BynSign({ size = 14, className, style }: Props) {
  const w = size * 1.4;
  return (
    <svg
      role="img"
      aria-label="белорусский рубль"
      width={w}
      height={size}
      viewBox="0 0 28 20"
      className={className}
      style={{ display: "inline-block", verticalAlign: "-0.15em", ...style }}
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Буква B — два округлых лепестка на вертикальной стойке */}
      <path
        d="M3 2 L3 18 M3 2.5 L10 2.5 C12.5 2.5 13.8 4 13.8 6.5 C13.8 8.8 12.4 10 10 10 L3 10 M3 10 L10.5 10 C13 10 14.4 11.4 14.4 14 C14.4 16.6 12.8 17.8 10.5 17.8 L3 17.8"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Буква r — короткая стойка с верхним крючком */}
      <path
        d="M17.5 8 L17.5 18 M17.5 10.5 C17.5 8.8 18.8 7.6 20.6 7.6 L21.8 7.6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Короткий горизонтальный штрих ПОД основанием B — валютный
          маркер. Длина ≈ ширина B, не пересекает r. */}
      <line
        x1="0.5"
        y1="19"
        x2="14.5"
        y2="19"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
    </svg>
  );
}
