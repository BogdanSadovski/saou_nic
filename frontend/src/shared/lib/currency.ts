/**
 * Currency formatting for the Belarusian ruble (BYN).
 *
 * BYN does not yet have a stable Unicode codepoint, so for the
 * graphical sign use the `<BynSign />` React component from
 * `@/shared/ui`. For plain-text contexts (toasts, URLs, alt text,
 * <select> values) use the textual fallback "Br" via `formatBYN`.
 */

/**
 * Format a numeric amount as a plain-text string with the "Br"
 * suffix. Use this in places that must be a string (toasts, log
 * messages, document.title etc.).
 *
 *     formatBYN(29) // "29 Br"
 *     formatBYN(1234.5) // "1 234,5 Br"
 */
export const formatBYN = (value: number): string => {
  const safe = Number.isFinite(value) ? value : 0;
  return `${safe.toLocaleString("ru-RU")} Br`;
};

/**
 * Format only the numeric portion (with ru-RU thousands separator).
 * Pair this with `<BynSign />` to render the modern graphical sign:
 *
 *     <>{formatBYNAmount(29)} <BynSign /></>
 */
export const formatBYNAmount = (value: number): string => {
  const safe = Number.isFinite(value) ? value : 0;
  return safe.toLocaleString("ru-RU");
};
