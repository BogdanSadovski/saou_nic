import type { CSSProperties } from "react";

/**
 * Универсальная иконка-аватар RealSync.
 *
 * Заменяет старые «инициалы» (СБ / IK / …) на единый ISO-куб с
 * лаймовой верхней гранью. Используется везде, где раньше показывали
 * 2-3 буквы — Navbar, Profile-карточка, кандидаты в админ-таблице,
 * имя в чате интервью.
 *
 * SVG inline (не <img>) — чтобы:
 *   • не зависел от загрузки картинки и не моргал
 *   • цвет фона/обводки наследовался от темы при желании
 *   • размер задавался через `size` или CSS-класс
 */
type Props = {
  size?: number;
  className?: string;
  style?: CSSProperties;
  /** ARIA-метка. По умолчанию декоративный. */
  alt?: string;
};

export function UserAvatar({ size = 36, className, style, alt }: Props) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 200 200"
      width={size}
      height={size}
      className={className}
      style={{ display: "block", borderRadius: "20%", ...style }}
      role={alt ? "img" : "presentation"}
      aria-label={alt}
      aria-hidden={alt ? undefined : true}
    >
      <rect width="200" height="200" rx="40" fill="#f5f1e8" />
      <defs>
        <pattern id="avatar-iso-grid" x="0" y="0" width="20" height="20" patternUnits="userSpaceOnUse">
          <line x1="0" y1="0" x2="20" y2="0" stroke="#8a857a" strokeWidth="0.6" opacity="0.45" />
          <line x1="0" y1="0" x2="0" y2="20" stroke="#8a857a" strokeWidth="0.6" opacity="0.45" />
        </pattern>
      </defs>
      <rect x="0" y="0" width="200" height="200" fill="url(#avatar-iso-grid)" />
      <g fill="none" stroke="#2b2a26" strokeWidth="3" strokeLinejoin="round">
        <polygon points="100,40 160,75 160,140 100,175 40,140 40,75" />
        <line x1="100" y1="40" x2="100" y2="105" />
        <line x1="40" y1="75" x2="100" y2="105" />
        <line x1="160" y1="75" x2="100" y2="105" />
      </g>
      <polygon points="100,40 160,75 100,105 40,75" fill="#cdee5a" opacity="0.9" />
    </svg>
  );
}

export default UserAvatar;
