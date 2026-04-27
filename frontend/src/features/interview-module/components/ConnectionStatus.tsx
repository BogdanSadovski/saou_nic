type Props = {
  connected: boolean;
  reconnectAttempts: number;
  lastError: string | null;
};

export function ConnectionStatus({ connected, reconnectAttempts, lastError }: Props) {
  const label = connected ? "Подключено" : "Переподключение";

  return (
    <div className={connected ? "connection-status connected" : "connection-status disconnected"}>
      <span>{label}</span>
      {!connected && reconnectAttempts > 0 ? <small>попытка #{reconnectAttempts}</small> : null}
      {lastError ? <small>{lastError}</small> : null}
    </div>
  );
}
