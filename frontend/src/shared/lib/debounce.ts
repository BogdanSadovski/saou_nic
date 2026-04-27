export function debounce<T extends (...args: never[]) => void>(
  fn: T,
  wait = 250,
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout> | null = null;

  return (...args: Parameters<T>) => {
    if (timeout) {
      clearTimeout(timeout);
    }

    timeout = setTimeout(() => {
      fn(...(args as never[]));
    }, wait);
  };
}
